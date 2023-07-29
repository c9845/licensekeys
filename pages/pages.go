/*
Package pages handles parsing the HTML templates used to build the GUI and returning
these templates when requested.

Most pages in the app are just a basic shell, with data being injected by API calls.
This was done so that pages are more reactive to filters and user activity. The API
calls are done via js/ts and Vue objects.

Most pages are returned via the App() or AppMapped() funcs. If a page needs to use,
validate, or handle a specific URL query parameter, inject some custom data, etc.
then a separate func in this package is defined. Since most pages are just HTML shells
with Vue/API calls populating pages, most pages use App() and AppMapped().

***

The below App and AppMapped funcs were defined to clean up having to define a separate
handler func for each basic page endpoint. For the pages that use App or AppMapped,
the handler funcs were near identical except the HTML template to show or the HTTP
router endpoint used. This resulted in a LOT of duplicate code defining handler funcs
all just to serve an HTML template. These funcs clean this up a lot.
*/
package pages

import (
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/c9845/hashfs"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/users"
)

// Config is the set of configuration settings for working with HTML templates. This
// also stores the parsed HTML templates after Build() is called and that can be shown
// with Show().
type Config struct {
	//Development is passed to each template when rendering the HTML to be sent to
	//the user so that the HTML can be altered based on if you are running your app
	//in a development mode/enviroment. Typically this is used to show a banner on
	//the page, loads extra diagnostic libraries or tools, and uses non-cache busted
	//static files.
	Development bool

	//UseLocalFiles is passed to each template when rendering the HTML to be sent to
	//the user so that the HTML can be altered to use locally hosted third party
	//libraries (JS, CSS) versus libraries retrieve from the internet.
	UseLocalFiles bool

	//Extension is the extension you use for your template files. The default is
	//"html".
	Extension string

	//TemplateFiles is the fs.FS filesystem that holds the HTML templates to be read
	//and used.
	TemplateFiles fs.FS

	//templates holds the list of parsed files constructed into golang templates.
	//
	//Templates are organized by subdirectory since that is how they are organized on
	//disk and this allows for filenames, or {{define}} blocks, to only need to be
	//unique within a subdirectory. This is where a specific template is looked up when
	//Show() is called to actually show and return the HTML to a user and their browser.
	//
	//Note that this only allows for a single tier of subdirectories, subs-of-subs are
	//not allowed!
	templates map[string]*template.Template

	//StaticFiles is the list of static files used to build the GUI (js, css, images).
	//This is a separate FS that is used for cache busting purposes. See the static()
	//func for more info.
	StaticFiles *hashfs.HFS

	//Debug prints out debugging information if true.
	Debug bool
}

// defaults
const (
	defaultExtension = ".html"
)

// cfg is the package level saved config. It is populated when you call Build().
var cfg Config

// Build parses each of the HTML files into a golang template, saves the parsed
// templates for future use, and saves the config. Build() must be called before Show()
// is called.
//
// Templates are build for the root directory in the given FS, as well as each sub-
// directory of the root (only one tier of subdirectories is built). Files from the
// root directory are inherited into each subdirectory, to allow for common
// files/partials (header, footer, html head, etc.) to be reused between subdirectories.
func (c *Config) Build() (err error) {
	//Some validation.
	if c.Extension == "" {
		c.Extension = defaultExtension
	}

	//Empty out map that holds built templates in case Build() is called more than
	//once.
	c.templates = make(map[string]*template.Template)

	//Parse root files. Root files can be used by themselves (via "" as the subdirectory
	//name to Show()) and be inherited into subdirectories for reuse.
	rootFilesPattern := path.Clean(path.Join(".", "*"+c.Extension))
	rootTemplates, err := template.New("").Funcs(funcMap).ParseFS(c.TemplateFiles, rootFilesPattern)
	if err != nil {
		return
	}
	c.templates[""] = rootTemplates

	if c.Debug {
		log.Println("pages.Build", "parsed root files")
	}

	//Parse each subdirectory, including root files.
	items, err := fs.ReadDir(c.TemplateFiles, ".")
	if err != nil {
		return
	}

	for _, i := range items {
		if !i.IsDir() {
			continue
		}

		subDirName := i.Name()
		subDirFilesPattern := path.Clean(path.Join(".", subDirName, "*"+c.Extension))
		patterns := []string{rootFilesPattern, subDirFilesPattern}
		subDirTemplates, innerErr := template.New("").Funcs(funcMap).ParseFS(c.TemplateFiles, patterns...)
		if innerErr != nil {
			err = innerErr
			return
		}
		c.templates[subDirName] = subDirTemplates

		if c.Debug {
			log.Println("pages.Build", "parsed subdirectory '"+subDirName+"' files")
		}
	}

	if c.Debug {
		log.Println("pages.Build", "Templates parsed for:")
		for k := range c.templates {
			log.Println(" - " + k)
		}
	}

	//Save the config, with now parsed templates, for future use.
	cfg = *c

	return
}

// Show renders a template as HTML and writes it to w. This works by taking a
// subdirectory's name and the name of a template file, looks up the template that
// was parsed earlier in Build(), and returns it with any injected data available at
// {{.Data}} in HTML templates.
//
// This does not work with {{defined}}{{end}} named templates. Templates must be files.
// However, a named template can include {{defined}}{{end}} and {{template}}{{end}}
// blocks.
//
// InjectedData is any custom data you want to return to use in the template, but
// is typically PageData{}. We allow for any type of data to be used though since it
// is more flexible.
//
// To serve files stored at the root of the templates fs.FS, use "" as the subdir.
func Show(w http.ResponseWriter, subdir, filename string, injectedData any) {
	//Organize data to render template. Some of the config data is provided for
	//debugging or development purposes.
	data := struct {
		Development   bool
		UseLocalFiles bool
		InjectedData  any
	}{
		Development:   cfg.Development,
		UseLocalFiles: cfg.UseLocalFiles,
		InjectedData:  injectedData,
	}

	//Add the extension to the template (file) name if needed. This handles instances
	//where Show() was called without the extension (which is semi-expected since it
	//shortens up the Show() call and removes the need to provide the extension each
	//time). We need the extension since that was the name of the file when it was
	//parsed to cache the templates.
	if filepath.Ext(filename) == "" {
		filename += cfg.Extension
	}

	//Serve the correct template based on the subdirectory. Remember, you could have
	//the same template name in multiple subdirectories!
	//
	//While we could return the error here (return errror.New...), we don't because
	//we assume that anyone developing using this package is acutely aware of their
	//subdirectory name(s) and will test this prior.
	t, ok := cfg.templates[subdir]
	if !ok {
		err := errors.New("pages.Show: invalid subdirectory '" + subdir + "'")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		//log errors out since they may not always show up in gui
		log.Println("pages.Show: invalid subdirectory", err)

		return
	}

	if err := t.ExecuteTemplate(w, filename, data); err != nil {
		//handle displaying of the templates if some kind of error occurs.
		http.Error(w, err.Error(), http.StatusNotFound)

		//log errors out since they may not always show up in gui
		log.Println("pages.Show: error during execute", err)

		return
	}
}

// PrintFSFileList prints out the list of files in an fs.FS.
//
// This is used ONLY for diagnostic/debug/development purposes and it typically paired
// with embedded files (go:embed).
func PrintFSFileList(e fs.FS) {
	//the directory "." means the root directory of the embedded file.
	const startingDirectory = "."

	err := fs.WalkDir(e, startingDirectory, func(path string, d fs.DirEntry, err error) error {
		log.Println(path)
		return nil
	})
	if err != nil {
		log.Fatalln("pages.PrintFSFileList", "error walking embedded directory", err)
		return
	}

	//exit after printing since you should never need to use this function outside of testing
	//or development.
	log.Println("pages.PrintFSFileList", "os.Exit() called, remove or skip PrintFSFileList to continue execution.")
	os.Exit(0)
}

// PageData is the format of our data we inject into a template to render it or show
// data. This struct is used so we always have a consistent format of data being
// injected into templates for easy parsing.
//
// This is provided to the injectedData field of Show().
//
// Any is used, instead of defined database types, so that we don't have import loops
// for ease of use.
type PageData struct {
	UserData    any //username, permissions, etc.
	AppSettings any //whether certain features are used or enabled.
	Data        any //misc. actual data we want to show in the gui.
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

	//Cache HTML in the browser since it rarely changes (app updates). This just makes
	//the GUI load a tiny bit quicker. Data for pages is loaded via API calls so this
	//just caches the base HTML.
	//
	//Pages with URL parameters (i.e.: ?lot=49112) will not be cached by browsers.
	setBaseHTMLCacheHeaders(w)

	//Serve the HTML template.
	Show(w, "app", last, pd)
}

// AppMapped serves a template from the "app" subdirectory of templates by matching up
// the full request URL path to an HTML template's filename. This is used for serving
// basic pages where no computations or functionality needs to be performed prior to
// showing the page, but where the last element in the path does not match the
// template's filename exactly (due to more complex path, url vs filename differences,
// etc.)
//
// For example: request is sent to "/licenses/reports/some-page/" but
// the HTML template is named "some-license-report-page". The path last element
// and name does not match because each fits and organization method based upon its
// use (HTTP router vs filesystem).
func AppMapped(w http.ResponseWriter, r *http.Request) {
	//List of URL paths to HTML filenames.
	m := map[string]string{
		"/licenses/add/": "create-license",

		"/activity-log/charts/activity-over-time-of-day/":   "activity-log-chart-over-time-of-day",
		"/activity-log/charts/max-avg-duration-per-month/":  "activity-log-chart-max-and-avg-monthly-duration",
		"/activity-log/charts/duration-of-latest-requests/": "activity-log-chart-latest-requests-duration",
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

	//Get basic page data we use for all HTML pages.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("Error getting page config data", err)
		return
	}

	//Cache HTML in the browser since it rarely changes (app updates). This just makes
	//the GUI load a tiny bit quicker. Data for pages is loaded via API calls so this
	//just caches the base HTML.
	setBaseHTMLCacheHeaders(w)

	//Serve the HTML template.
	Show(w, "app", filename, pd)
}

// setBaseHTMLCacheHeaders sets the headers to tell a user's browser to cache the base
// HTML for pages. This should only be used for serving pages from the pages.App,
// pages.AppMapped and help pages since there isn't any page-specific data injected
// into the HTML via golang templating.
//
// This was functionized so we only have to change the cache time in one place.
func setBaseHTMLCacheHeaders(w http.ResponseWriter) {
	//Hours to cache HTML in user's browsers.
	const hours = 1

	//Set headers.
	w.Header().Add("Cache-Control", "no-transform,public,max-age="+strconv.Itoa(60*60*hours))
}
