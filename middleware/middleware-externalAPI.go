/*
Package middleware handles authentication, user permissions, and any other tasks
that occur with a request to this app.

This file handles external api access to this app using an API key.
*/
package middleware

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/c9845/licensekeys/apikeys"
	"github.com/c9845/licensekeys/db"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v2"
)

//errAPIAccessNotAllowed is used when a request is made using and API key but the app
//is configured to not allow api access. This is a setting in app settings.
var errAPIAccessNotAllowed = errors.New("api access it not enabled")

//ExternalAPI handles authenticating access to the public endpoints using api keys.
func ExternalAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Get API key.
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

		//Check if api access is enabled.
		appData, err := db.GetAppSettings(r.Context())
		if err != nil {
			output.Error(err, "Could not look up app settings to verify if api is enabled.", w)
			return
		}
		if !appData.AllowAPIAccess {
			output.Error(errAPIAccessNotAllowed, "API access has not been enabled.", w)
			return
		}

		//Calidate the key.
		cols := sqldb.Columns{
			db.TableAPIKeys + ".Active",
			db.TableAPIKeys + ".Description",
		}
		keyData, err := db.GetAPIKeyByKey(r.Context(), key, cols)
		if err == sql.ErrNoRows {
			output.ErrorInputInvalid("The API key you provided does not exist.", w)
			return
		} else if err != nil {
			output.Error(err, "Could not authenticate using api key.", w)
			return
		}

		//Make sure key is active.
		if !keyData.Active {
			log.Println("Inactive or invalid API key", keyData.Description)
			output.ErrorInputInvalid("The API key you provided is inactive or invalid.", w)
			return
		}

		//Make sure request is for a valid public endpoint.
		err = apikeys.IsPublicEndpoint(r.URL.Path)
		if err != nil {
			log.Println("middleware.externalAPI, endpoint not public", r.URL.Path, keyData.Description)
			output.Error(err, "You cannot access this endpoint via the API.", w)
			return
		}

		//Diagnostic logging.
		log.Println("API Access", r.URL.Path, "via key:", keyData.Description)

		//move to next middleware or handler
		next.ServeHTTP(w, r)
	})
}
