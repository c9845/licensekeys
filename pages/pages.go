/*
Package pages handles parsing the HTML templates used to build the GUI and returning
these templates when requested.

Most pages in the app are just a basic shell, with data being injected by API calls.
This was done so that pages are more reactive to filters and user activity. The API
calls are done via js/ts and Vue objects.

Most pages are returned via the Page() or Show() funcs. Page is used for most
pages, see the router in main.go. Show() is used when Page() isn't used and we
defined a special http handler for the endpoint. Show() is when we aren't just
returning a base HTML page and we are doing some validation or gathering data to
inject into a template with Golang templating.
*/
package pages

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/c9845/hashfs"
	"github.com/c9845/licensekeys/v3/db"
	"github.com/c9845/licensekeys/v3/users"
)

// Config is the set of configuration settings for working with HTML templates. This
// also stores the parsed HTML templates after ParseTemplates() is called.
type Config struct {
	//Development is passed to each template when rendering the HTML to be sent to
	//the user so that the HTML can be altered based on if you are running your app
	//in a development mode/enviroment.
	//
	//Typically this is used to show a banner on the page, loads extra diagnostic
	//third-party libraries (Vue with devtools support), and uses non-cache busted
	//static files.
	Development bool

	//UseLocalFiles is passed to each template when rendering the HTML to be sent to
	//the user so that the HTML can be altered to use locally hosted third party
	//libraries (JS, CSS) versus libraries retrieve from the internet.
	//
	//This is typically set via config file field.
	UseLocalFiles bool

	//Extension is the extension you use for your template files. The default is
	//".html".
	Extension string

	//TemplateFiles is the fs.FS filesystem that holds the HTML templates to be read
	//and used. This is from os.DirFS or embed.
	TemplateFiles fs.FS

	//StaticFiles is the list of static files used to build the GUI (js, css, images).
	//This is a separate FS that is used for cache busting purposes. See the static()
	//func for more info.
	StaticFiles *hashfs.HFS

	//templates holds the list of parsed files constructed into Golang templates.
	templates *template.Template

	//Debug prints out debugging information if true.
	Debug bool
}

// defaults
const (
	defaultExtension = ".html"
)

// cfg is the package level saved config. It is populated when you call
// ParseTemplates().
var cfg Config

// ParseTemplates parses HTML templates from the config's TemplateFiles and saves
// the parsed templates for use with Show().
func (c *Config) ParseTemplates() (err error) {
	//Some validation.
	if c.Extension == "" {
		c.Extension = defaultExtension
	}

	//Make sure this func is only called once.
	if c.templates != nil {
		log.Fatalln("pages.ParseTemplates has already been called, it can only be called once")
		return
	}

	//Parse the templates.
	t, err := c.parseTemplates()
	if err != nil {
		return
	}

	//Diagnostics.
	if c.Debug {
		//Not use .DefinedTemplates() here because the data is a bit messy. Our
		//method below is cleaner and more organized.

		x := t.Templates()
		names := []string{}
		for _, y := range x {
			names = append(names, y.Name())
		}

		log.Println("##################################################################")
		sort.Strings(names)
		for _, s := range names {
			log.Println(s)
		}
	}

	//Save the config, with now parsed templates, for future use.
	c.templates = t
	cfg = *c
	return nil
}

// parseTemplates parses the templates noted in the fs.FS in the config by walking
// the fs.FS recursively.
//
// The c.TemplatesFiles fs.FS is "inside" the website/ directory. I.e., the
// subdirectories of c.TemplateFiles are root, static, templates.
//
// We don't use ParseFS() here because that doesn't allow us to have the same
// template or filename multiple times. Using Parse() allows us to have duplicate
// names (think app vs help docs) because we name the template using the path to
// the template, not just the filename. We used to use ParseFS() in conjunction
// with a single-level subdirectory handling, but it created a mess of files in
// one subdirectory that were organized by prefixed filenames that then required
// a mapping func to map URL endpoints to files.
func (c *Config) parseTemplates() (t *template.Template, err error) {
	//Define a blank template store to start working with. As we parse templates
	//in WalkDir below, the parsed templates will be added to this same template
	//store
	//
	//Always import our extra funcs.
	t = template.New("").Funcs(funcMap)

	//Parse the templates by walking the fs.FS recursively.
	err = fs.WalkDir(c.TemplateFiles, ".", func(p string, d fs.DirEntry, err error) error {
		//Handle odd path errors.
		if err != nil {
			return err
		}

		//Don't parse directory listing as a template, for obvious reasons.
		if d.IsDir() {
			return nil
		}

		//Ignore non-template files. Template files end in a known provided extension,
		//most likely .html.
		if path.Ext(p) != c.Extension {
			return nil
		}

		if c.Debug {
			log.Println("walking path...", p, "...found template file")
		}

		//Read the template file from the fs.FS.
		f, err := fs.ReadFile(c.TemplateFiles, p)
		if err != nil {
			return err
		}

		//Create name for template based on the file's path in the fs.FS. This name
		//will be used to reference the template in the template store. This name is
		//what we will refer to when we want to show this template in Show(). This
		//name will most likely match the URL endpoint being requested.
		//
		//This name must be unique among all parsed templates! This should not be an
		//issues since the filename/path must be unique already (the OS does not allow
		//the same filename at the same path!).
		//
		//The name CANNOT start with a "/" because we are using an fs.FS.
		//See https://pkg.go.dev/io/fs@master#ValidPath.
		name := p

		//Parse the template.
		_, err = t.New(name).Funcs(funcMap).Parse(string(f))
		if err != nil {
			return err
		}

		if c.Debug {
			log.Println(p, "stored as", name)
		}

		//Done, continue walking directory tree.
		return nil
	})

	return
}

// show renders a template as HTML and writes it to w.
//
// You should most likely be using Show() instead.
func show(t *template.Template, w http.ResponseWriter, templateName string, injectedData any) {
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

	//Clean to make sure we don't have trailing slash before we append file extension.
	templateName = path.Clean(templateName)

	//Catch if a template starts with a slash, which is not allowed. Log this for
	//fixing even though we fix it here.
	if strings.HasPrefix(templateName, "/") {
		log.Println("pages.Show", "template name must not start with a leading slash; slash automatically removed", templateName)
		templateName = strings.TrimPrefix(templateName, "/")
	}

	//Add the extension to the template (file) name if needed. This handles instances
	//where Show() was called without the extension (which is semi-expected since it
	//shortens up the Show() call and removes the need to provide the extension each
	//time). We need the extension since that was the name of the file when it was
	//parsed to cache the templates.
	if path.Ext(templateName) == "" {
		templateName += cfg.Extension
	}

	//Make sure a template with the name p exists, or check if a template files exists
	//in a subdirectory.
	//
	//This handles something like "index.html" existing in a subdirectory and index
	//being served if the path to the directory is given. Except, we don't want to
	//use files named "index.html" because it makes finding or opening files harded
	//in development (many index.html files).
	exists := t.Lookup(templateName)
	if exists == nil {
		//Create the name to the "index" template, which is simply the end of the
		//provided path doubled (last directory and filename are the same).
		//ex:
		// - /app/activity-log.html -> /app/activity-log/activity-log.html
		index := strings.TrimSuffix(templateName, path.Ext(templateName))
		index = path.Join(index, path.Base(index))
		index += path.Ext(templateName)
		index = path.Clean(index)

		if cfg.Debug {
			log.Println("pages.Show", "template at path could not be found, trying in same-named subdirectory")
			log.Println(" before:", templateName)
			log.Println(" after: ", index)
		}

		exists = t.Lookup(index)
		if exists == nil {
			log.Fatalln("pags.Show", "could not find template", index)
			return
		}

		templateName = index
	}

	err := t.ExecuteTemplate(w, templateName, data)
	if err != nil {
		//handle displaying of the templates if some kind of error occurs.
		http.Error(w, err.Error(), http.StatusNotFound)

		//log errors out since they may not always show up in gui
		log.Println("pages.Show", "error during execute", err)

		return
	}
}

// Show renders a template as HTML and writes it to w. ParseTemplates() must have
// been called first. injectedData is available within the template at the .Data field.
// templateName should be the path to a template file such as /app/suppliers.html.
//
// The path must match a parsed template name. AKA, does the URL being requested match
// a file within the website/templates/ directory.
//
// If you don't need to inject any data into the page, call Page() instead.
func Show(w http.ResponseWriter, templateName string, injectedData any) {
	show(cfg.templates, w, templateName, injectedData)
}

// Page builds and returns a template to w based on the URL path provided in r. This
// func is used directly in r.Handle() in main.go for pages that don't require any
// page-specific data to be injected into them. If you need to inject page-specific
// data, call Show() directly.
func Page(w http.ResponseWriter, r *http.Request) {
	//Get path.
	//
	//Clean will removed the trailing slash.
	//ex.: /app/company/ -> /app/company
	templateName := path.Clean(r.URL.Path)
	templateName = strings.TrimPrefix(templateName, "/")

	//Debug logging.
	if cfg.Debug {
		log.Println("pages.Page", "request:", r.URL.Path, "-- cleaned:", templateName)
	}

	//If we are showing a help page, just display the page. We don't need to
	//get any data from the database or session for help pages.
	if strings.Contains(templateName, "/help/") {
		Show(w, templateName, nil)
		return
	}

	//User is requesting a logged-in page. Get data to build page properly.
	//Mostly, we need user permissions to show/hide certain elements.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("pages.Page", "error getting page config data", err)
		return
	}

	Show(w, templateName, pd)
}

// PageData is the format of our data that we inject into a template. This struct is
// used so we always have a consistent format of data being injected into templates
// for easy parsing.
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
	u, err := users.GetUserDataFromRequest(r)
	if err != nil {
		return
	}

	//Remove secrets! Since GetUserDataFromRequest() looks up/returns all columns (*),
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
