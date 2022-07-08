package pages

import (
	"log"
	"net/http"
	"strconv"

	"github.com/c9845/licensekeys/config"
	"github.com/c9845/licensekeys/db"
	"github.com/c9845/templates"
)

//Users shows the page to manage users.
func Users(w http.ResponseWriter, r *http.Request) {
	//Get data to build gui.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("Error getting page config data", err)
		return
	}

	//Get the min password length to show in gui and use for validation.
	data := struct {
		MinPasswordLength int
	}{config.Data().MinPasswordLength}
	pd.Data = data

	templates.Show(w, "app", "users", pd)
}

//UserLogins shows the page of user logins to the app. You can also filter by user ID.
func UserLogins(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)
	rows, _ := strconv.ParseInt(r.FormValue("rows"), 10, 64)

	//Validate.
	if userID < 0 {
		userID = 0
	}
	if rows < 0 {
		rows = 200
	}

	//Get results.
	logins, err := db.GetUserLogins(r.Context(), userID, uint16(rows))
	if err != nil {
		e := ErrorPage{
			PageTitle:   "View User Logins",
			Topic:       "An error occured while trying to display the login log.",
			Message:     "The log data cannot be displayed.",
			Solution:    "Please contact an administrator and have them look at the logs to investigate.",
			ShowLinkBtn: true,
		}
		log.Println("pages.UserLogins", "Could not look up list of logins", err)

		ShowError(w, r, e)
		return
	}

	//Get data to build gui.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("Error getting page config data", err)
		return
	}

	//Show page.
	pd.Data = logins
	templates.Show(w, "app", "user-logins", pd)
}
