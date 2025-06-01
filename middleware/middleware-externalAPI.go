package middleware

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/c9845/licensekeys/v4/apikeys"
	"github.com/c9845/licensekeys/v4/db"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
)

/*
This file checks if an API key was provided, and is valid, when an API request is
made to a publically accessible endpoint.
*/

var (
	// errAPIAccessNotAllowed is used when a public API endpoint is accessed but API
	// access has not been allowed in the App Settings.
	errAPIAccessNotAllowed = errors.New("api access is not enabled")

	// errNonPublicEndpoint is returned when a request is made with an API key to a
	//URL that is not publicly accessible.
	errNonPublicEndpoint = errors.New("api access denied to non-public endpoint")
)

// ExternalAPI is used to verify a request to an publically accessible endpoint is
// being made with a valid API key. If the API key is valid, the request is redirected
// to the next HTTP handler, otherwise an error message is returned.
//
// This func should be called upon every publically accessible endpoint.
func ExternalAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Check if the public API is enabled in the App Settings.
		as, err := db.GetAppSettings(r.Context())
		if err != nil {
			output.Error(err, "Could not look up App Settings to verify if the public API is enabled.", w)
			return
		}
		if !as.AllowAPIAccess {
			output.Error(errAPIAccessNotAllowed, "The public API has not been enabled.", w)
			return
		}

		//Get the API key from the request. The API key can be provided via a few
		//methods:
		//   1) The Authorization header, with the Bearer scheme.
		//   2) The apiKey as a URL parameter (legacy, don't use).
		key := ""
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			//Header was provided, but we don't know if Bearer scheme was provided
			//nor do we know if an actual API key was provided.

			//Make sure correct scheme was provided in header.
			if !strings.Contains(authHeader, "Bearer") {
				p := output.Payload{
					OK:   false,
					Type: "unauthorized",
					ErrorData: output.ErrorPayload{
						Error:   "invalid authorization header scheme",
						Message: "An API was not provided with the Bearer scheme.",
					},
				}
				w.Header().Set("WWW-Authenticate", "Bearer")
				output.Send(p, w, http.StatusUnauthorized)
				return
			}

			//Make sure an API key was provided.
			k := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
			if k == "" {
				p := output.Payload{
					OK:   false,
					Type: "unauthorized",
					ErrorData: output.ErrorPayload{
						Error:   "missing api key",
						Message: "An API key was not provided via the Authorization header (using the Bearer scheme).",
					},
				}
				w.Header().Set("WWW-Authenticate", "Bearer")
				output.Send(p, w, http.StatusUnauthorized)
				return
			}

			//Save key found in header for further validation and use.
			key = k

		} else {
			//Header was NOT provided. Check if the API key was provided via a URL
			//parameter (legacy!).
			k := strings.TrimSpace(r.FormValue("apiKey"))
			if k == "" {
				p := output.Payload{
					OK:   false,
					Type: "unauthorized",
					ErrorData: output.ErrorPayload{
						Error:   "missing api key",
						Message: "An API key was not provided via the Authorization header.",
					},
				}
				w.Header().Set("WWW-Authenticate", "Bearer")
				output.Send(p, w, http.StatusUnauthorized)
				return
			}

			if len(k) != apikeys.KeyLength() {
				p := output.Payload{
					OK: false,
					// Type: "unauthorized",
					ErrorData: output.ErrorPayload{
						Error:   "invalid api key",
						Message: "The API key you provided was not formatted correctly.",
					},
				}
				w.Header().Set("WWW-Authenticate", "Bearer")
				output.Send(p, w, http.StatusUnauthorized)
				return
			}

			//Save key found in URL for further validation and use.
			key = k

			//Set response header so that users know to use Authorization header
			//instead of URL parameter for providing API key in the future.
			w.Header().Set("WWW-Authenticate", "Bearer")
			log.Println("middleware.ExternalAPI", "Providing API key via URL parameter is deprecated, use Authorization header (Bearer scheme).")
		}

		//Validate the API Key exists and get its data.
		cols := sqldb.Columns{db.TableAPIKeys + ".*"}
		keyData, err := db.GetAPIKeyByKey(r.Context(), key, cols)
		if err == sql.ErrNoRows {
			p := output.Payload{
				OK:   false,
				Type: "unauthorized",
				ErrorData: output.ErrorPayload{
					Error:   "invalid api key",
					Message: "The API key you provided is does not exist.",
				},
			}
			w.Header().Set("WWW-Authenticate", "Bearer")
			output.Send(p, w, http.StatusUnauthorized)
			return
		} else if err != nil {
			output.Error(err, "Could not validate the API key you provided.", w)
			return
		}

		//Make sure the API Key is active.
		if !keyData.Active {
			p := output.Payload{
				OK:   false,
				Type: "unauthorized",
				ErrorData: output.ErrorPayload{
					Error:   "invalid api key",
					Message: "The API key you provided is no longer active.",
				},
			}
			w.Header().Set("WWW-Authenticate", "Bearer")
			output.Send(p, w, http.StatusUnauthorized)
			return
		}

		//Make sure request is for a valid public endpoint and that the API key has
		//permission for the endpoint.
		//
		//Only some functionality is available via the public API to prevent any data
		//corruption or other broken functionality.
		//
		//This list of endpoints must match the list defined in main.go router.
		//
		//Permissions are checked here, not within each endpoint's handler, because
		//we don't check user permissions in each endpoint handler either, we check
		//them in middelware. This keeps permissions checking out of the endpoint
		//handlers and allows the endpoint handlers to be easily reused for private
		//within-app user-session based authentication or public api-key based auth.
		err = nil
		switch r.URL.Path {
		case "/api/v1/licenses/add/":
		case "/api/v1/licenses/download/":
		case "/api/v1/licenses/renew/":
		case "/api/v1/licenses/disable/":
		default:
			output.Error(errNonPublicEndpoint, "You cannot access this endpoint via the public API.", w)
			return
		}

		//Save API key ID to context for use further in this request. For example, this
		//is used to save to the activity log when this request is completed.
		ctx := context.WithValue(r.Context(), apikeys.APIKeyContextKey, keyData.ID)
		r = r.WithContext(ctx)

		//Diagnostic logging.
		log.Println("API Access", r.URL.Path, "Via Key:", keyData.Description)

		//Move to next middleware or handler.
		next.ServeHTTP(w, r)
	})
}
