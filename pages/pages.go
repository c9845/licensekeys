package pages

import (
	"database/sql"
	"log"
	"net/http"
	"path"
	"strconv"

	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/users"
	"github.com/c9845/sqldb/v2"
	"github.com/c9845/templates"
)

// PageData is the format of our data we inject into a template to render it or show
// data. This struct is used so we always have a consistent format of data being
// injected into templates for easy parsing.
//
// Returning interface{} instead of defined db types for ease of use.
type PageData struct {
	UserData    interface{} //username, permissions, etc.
	AppSettings interface{} //whether certain features are used or enabled.
	Data        interface{} //misc. actual data we want to show in the gui.
}

// getPageConfigData gets the common data needed to build pages. This retrieves the
// common data we use on every page, such as user data (mostly for permissions and
// hiding/showing certain elements), app settings (for hiding/showing certain elements
// or functionality), and other common stuff like company and license data. This is used
// on nearly every page requested/shown to fill the PageData struct (except the Data
// field).
func getPageConfigData(r *http.Request) (pd PageData, err error) {
	//Get user data.
	u, err := users.GetUserDataByRequest(r)
	if err != nil {
		return
	}

	//Remove secrets! Since GetUserDataByRequest() looks up/returns all columns (*),
	//we want to remove some data since it is sensitive for security.
	u.Password = ""
	u.TwoFactorAuthSecret = ""

	//Get app settings.
	as, err := db.GetAppSettings(r.Context())
	if err != nil {
		log.Println("pages.getPageConfigData", "Could not look up app settings.", err)
		return
	}

	//Save data to build page.
	pd.UserData = u
	pd.AppSettings = as
	return
}

// Main shows the main GUI of the license server app. The main page shows
// navigation links to create & manage your apps, create & manage licenses,
// manage users, manage this app, and view activity logging.
func Main(w http.ResponseWriter, r *http.Request) {
	//get data to build gui
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("Error getting page config data", err)
		return
	}

	templates.Show(w, "app", "main", pd)
}

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

	//get data to build gui
	//Send back the license ID to embed in HTML hidden input so it can be read
	//by Vue to use in api calls.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("Error getting page config data", err)
		return
	}
	pd.Data = licenseID

	//show page
	templates.Show(w, "app", "license", pd)
}

/*
*************************************************************************************
***
The below App and AppMapped funcs were defined to clean up having to define a separate
handler func for each basic page endpoint. For the pages that use App or AppMapped,
the handler funcs were near identical except the HTML template to show or the HTTP
router endpoint used. This resulted in a LOT of duplicate code defining handler funcs
all just to serve an HTML template. These funcs clean this up a lot.

Reference commit from 6/24/22 from other projects.
***
*************************************************************************************
*/

// App serves a template based upon the last piece of the path exactly matching an HTML
// template's filename. This is used for serving basic pages where no computations
// or functionality needs to be performed prior to showing the page.
//
// You would not use this when you need to provide more data to an HTML template, need
// to perform some kind of computation or validation prior to showing a template, or
// if you are serving the filename on a different URL path.
func App(w http.ResponseWriter, r *http.Request) {
	//Get last element of path.
	u := r.URL.Path
	last := path.Base(u)

	//Get basic page data we use for all HTML pages.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("Error getting page config data", err)
		return
	}

	//Serve the HTML template.
	templates.Show(w, "app", last, pd)
}

// AppMapped serves a template based upon matching up the full request URL path to an
// HTML template's filename. This is used for serving basic pages where no computations
// or functionality needs to be performed prior to showing the page, but where the last
// element in the path does not match the template's filename exactly.
//
// Mapping a URL path to a filename is necessary to handle the different organization
// structures for HTML templates in the filesystem and the URL paths for HTTP routing.
//
// For example: request is sent to "/raw-materials/reports/underused-containers/" but
// the HTML template is named "raw-material-underused-containers". The path last element
// and name does not match because each fits and organization method based upon its
// use (HTTP router vs filesystem).
func AppMapped(w http.ResponseWriter, r *http.Request) {
	//List of URL paths to HTML filenames.
	m := map[string]string{
		"/licenses/add/": "create-license",
	}

	//Get the path being accessed.
	path := r.URL.Path

	//Get matching filename.
	filename, ok := m[path]
	if !ok {
		ep := ErrorPage{
			PageTitle: "Unable to Find Path",
			Topic:     "Could not find file to build page with.",
			Solution:  "Please contact an administrator and tell them what page you were trying to visit.",
		}
		ShowError(w, r, ep)
		return
	}

	log.Println("pages.AppMapped", path, filename)

	//Get basic page data we use for all HTML pages.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("Error getting page config data", err)
		return
	}

	log.Println("pages.AppMapped", path, filename)

	//Serve the HTML template.
	templates.Show(w, "app", filename, pd)
}
