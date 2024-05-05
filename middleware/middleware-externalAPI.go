package middleware

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"slices"
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

	//errAPIKeyNotAuthorized is used when an API Key is active, but it does not have
	//permission to perform the action at the endpoint.
	errAPIKeyNotAuthorized = errors.New("api key does not have required permission")

	// errNonPublicEndpoint is returned when a request is made with an API key to a
	//URL that is not publicly accessible.
	errNonPublicEndpoint = errors.New("api access denied to non-public endpoint")
)

// publicEndpoints are the list of URLs a user can access via an API key. This list
// is checked against in middleware to make sure a request using an API key is
// accessing a publicly accessible endpoint.
var publicEndpoints = []string{
	"/api/v1/licenses/add/",
	"/api/v1/licenses/download/",
	"/api/v1/licenses/renew/",
	"/api/v1/licenses/disable/",
}

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

		//Make sure request is for a valid public endpoint.
		if slices.Contains(publicEndpoints, r.URL.Path) {
			output.Error(errNonPublicEndpoint, "You cannot access this endpoint via the API.", w)
		}

		//Diagnostic logging.
		log.Println("API Access", r.URL.Path, "Via Key:", keyData.Description)

		//Move to next middleware or handler.
		next.ServeHTTP(w, r)
	})
}
