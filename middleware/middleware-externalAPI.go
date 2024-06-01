package middleware

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/c9845/licensekeys/v2/apikeys"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
)

// This file handles external API access to this app using an API key.

var (
	// errAPIAccessNotAllowed is used when a public API endpoint is accessed but API
	// access has not been allowed in the App Settings.
	errAPIAccessNotAllowed = errors.New("api access is not enabled")

	// errNonPublicEndpoint is returned when a request is made with an API key to a
	//URL that is not publicly accessible.
	errNonPublicEndpoint = errors.New("api access denied to non-public endpoint")
)

// ExternalAPI handles authenticating access to the public endpoints using api keys.
func ExternalAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Get API key from request.
		key := strings.TrimSpace(r.FormValue("apiKey"))

		//Basic validation.
		if key == "" {
			output.ErrorInputInvalid("No API Key was provided.", w)
			return
		} else if len(key) != apikeys.KeyLength() {
			output.ErrorInputInvalid("The API Key you provided is not valid.", w)
			// log.Println("Provided API Key:", key, len(key), apikeys.KeyLength())
			return
		}

		//Check if API access is enabled.
		as, err := db.GetAppSettings(r.Context())
		if err != nil {
			output.Error(err, "Could not look up app settings to verify if api is enabled.", w)
			return
		}
		if !as.AllowAPIAccess {
			output.Error(errAPIAccessNotAllowed, "API access has not been enabled.", w)
			return
		}

		//Validate the API Key exists and get its data.
		cols := sqldb.Columns{db.TableAPIKeys + ".*"}
		keyData, err := db.GetAPIKeyByKey(r.Context(), key, cols)
		if err == sql.ErrNoRows {
			output.ErrorInputInvalid("The API key you provided does not exist.", w)
			return
		} else if err != nil {
			output.Error(err, "Could not authenticate using api key.", w)
			return
		}

		//Make sure the API Key is active.
		if !keyData.Active {
			log.Println("Inactive or invalid API key", keyData.Description)
			output.ErrorInputInvalid("The API key you provided is inactive or invalid.", w)
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
			output.Error(errNonPublicEndpoint, "You cannot access this endpoint via the API.", w)
			return
		}

		//Diagnostic logging.
		log.Println("API Access", r.URL.Path, "Via Key:", keyData.Description)

		//Move to next middleware or handler.
		next.ServeHTTP(w, r)
	})
}
