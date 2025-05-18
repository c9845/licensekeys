package pages

import (
	"log"
	"net/http"

	"github.com/c9845/licensekeys/v3/config"
)

//This file specifically handles user related pages. This functionality was broken
//out into a separate file mostly to handle the password length requirements from the
//config file.

// Users shows the page to manage users.
//
// This does not use Page() since we need to get the minimum password length to build
// the GUI with.
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

	//Show page.
	Show(w, "app/administration/users.html", pd)
}

// UserProfile shows the page where a user can view and manage their own user account
// or profile. For now, user's can only change their passwords since we do not want
// them to be able to change permissions (for obvious reasons) or alerts (so they get
// alerts for what admin's deem important).
//
// This does not use Page() since we need to get the minimum password length to build
// the GUI with.
func UserProfile(w http.ResponseWriter, r *http.Request) {
	//Get basic data to build GUI.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("Error getting page config data", err)
		return
	}

	//Get the minimum password length to show in GUI and use for validation.
	data := struct {
		MinPasswordLength int
	}{config.Data().MinPasswordLength}
	pd.Data = data

	//Show page.
	Show(w, "app/user-profile.html", pd)
}
