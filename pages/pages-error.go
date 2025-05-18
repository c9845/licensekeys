package pages

import (
	"log"
	"net/http"

	"github.com/c9845/licensekeys/v3/db"
)

//This file defines handling for general error pages throughout the app. This page
//shows a general error message with a possible fix to users, but doesn't show low-
//level information about an error.

// ErrorPage is the data used to show messages when an error or unexpected event occurs
// this stuff is injected into a template to describe to user what occured and how to
// fix it.
type ErrorPage struct {
	PageTitle string //card header

	//These may simply be concatted in the gui but makes code cleaner and more easily reused.
	Topic    string //general topic of the error
	Message  string //what error occured
	Solution string //how to fix or work around the error

	ShowLinkBtn bool   //whether or not a footer button with link is shown
	Link        string //url to use for link, if blank will default to /app/ (main page).
	LinkText    string //text to show on button, if blank will default to Main Menu.
}

// ShowError shows the error page with injected data. This this func cleans up the code
// whereever an error page is called/shown since it is common throughout the app. The
// first part of this func is similar to getPageConfigData but allows for skipping the
// user data since we do not use it on the error page.
func ShowError(w http.ResponseWriter, r *http.Request, ep ErrorPage) {
	//Get app settings.
	as, err := db.GetAppSettings(r.Context())
	if err != nil {
		log.Println("pages.getPageConfigData", "Could not look up app settings.", err)
		return
	}

	//Build data object to build page with
	pd := PageData{
		AppSettings: as,
		Data:        ep,
	}

	//Show the page.
	Show(w, "app/error-page.html", pd)
}
