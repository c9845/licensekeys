package pages

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/c9845/licensekeys/v3/db"
	"github.com/c9845/sqldb/v3"
)

//This file specifically handles page that shows the data for a specific license that
//has been generated. This functionality was broken out into a separate file because
//there is some extra validation before the page is displayed.

// License shows the data for a specific license.
func License(w http.ResponseWriter, r *http.Request) {
	//make sure a license ID was provided and it is valid.
	licenseID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if licenseID < 1 {
		e := ErrorPage{
			PageTitle:   "View License",
			Topic:       "The license ID provided is not valid.",
			Solution:    "Please choose a license from the list of licenses.",
			ShowLinkBtn: true,
			Link:        "/licenses/",
			LinkText:    "Licenses",
		}
		ShowError(w, r, e)
		return
	}

	cols := sqldb.Columns{db.TableLicenses + ".Active"}
	_, err := db.GetLicense(r.Context(), licenseID, cols)
	if err == sql.ErrNoRows {
		e := ErrorPage{
			PageTitle:   "View License",
			Topic:       "The license ID provided is does not exist.",
			Solution:    "Please choose a license from the list of licenses.",
			ShowLinkBtn: true,
			Link:        "/licenses/",
			LinkText:    "Licenses",
		}
		ShowError(w, r, e)
		return
	} else if err != nil {
		e := ErrorPage{
			PageTitle:   "View License",
			Topic:       "An error occured while looking up this license's data.",
			Message:     err.Error(),
			Solution:    "Please contact an administator and have them investigate this error.",
			ShowLinkBtn: true,
		}
		ShowError(w, r, e)
		return
	}

	//Get data to build GUI.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("Error getting page config data", err)
		return
	}

	//Send back the license ID to embed in HTML hidden input so it can be read
	//by Vue to use in api calls.
	//
	//TODO: remove this, just get license ID from URL via JS.
	pd.Data = licenseID

	//show page
	Show(w, "app/licensing/license.html", pd)
}
