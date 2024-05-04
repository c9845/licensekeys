package pages

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/sqldb/v3"
)

//This file specifically handles the main page of the app where users can navigate
//from. This functionality was broken out into a separate file just for organization
//purposes.

// Main shows the main page of the app.
func Main(w http.ResponseWriter, r *http.Request) {
	//Get basic page data we use for all HTML pages.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("Error getting page config data", err)
		return
	}

	//Check if default admin user is enabled. This user should not be enable since
	//the password is set to a default value when app is deployed. This password
	//should be changed, user should be disabled, and all permissions should be
	//turned off. Display warning for users to do this.
	cols := sqldb.Columns{db.TableUsers + ".Active"}
	u, err := db.GetUserByUsername(r.Context(), db.InitialUserUsername, cols)
	if err != sql.ErrNoRows && err != nil {
		log.Println("pages.Main", "could not look up default initial user to verify user is disabled, ignoring error", err)
		//ignore error since this isn't the end of the world, plus we told user in docs and install to disable this user
		//u.Active will be false (the default value) so warning won't be shown in gui
	}

	//Build data to return.
	data := struct {
		IsInitialDefaultUserActive bool
	}{
		IsInitialDefaultUserActive: u.Active,
	}
	pd.Data = data

	//Serve the HTML template.
	Show(w, "/app.html", pd)
}
