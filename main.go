/*
This app creates and manages licence keys for software applications. Each license
is signed by a private key where the matching public key is embedded in your
application. This package also provides code for validating the license key in your
applications (obviously, golang only apps).
*/
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/c9845/hashfs"
	"github.com/c9845/licensekeys/v2/activitylog"
	"github.com/c9845/licensekeys/v2/apikeys"
	"github.com/c9845/licensekeys/v2/apps"
	"github.com/c9845/licensekeys/v2/appsettings"
	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/customfields"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/keypairs"
	"github.com/c9845/licensekeys/v2/license"
	"github.com/c9845/licensekeys/v2/middleware"
	"github.com/c9845/licensekeys/v2/pages"
	"github.com/c9845/licensekeys/v2/users"
	"github.com/c9845/licensekeys/v2/version"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

// embed HTML templates and static files into app.
// This is done so that we don't need to distribute our HTML, CSS, JS, etc. files
// separately. The end-user will not have to set the WebFilesPath field in their config
// file.
//
// Embedding files will increase the size of the built executable. To limit the size of
// the executable, we only embed the necessary files. This ends up being a lot of files
// since we still want to allow the end-user to serve third party files from the local
// server versus from over the internet/CDN.
//
//go:embed website/templates/*
//go:embed website/static/css
//go:embed website/static/js/*.js
//go:embed website/static/js/vendor
//go:embed website/root/*
var embeddedFiles embed.FS

// Vars for handing files stored on-disk or embedded.
var sourceFilesFS fs.FS           //files from the website/ directory.
var templateFilesFS fs.FS         //subdirectory of sourceFilesFS for HTML templates.
var staticFilesFS fs.FS           //subdirectory of sourceFilesFS for static files (js, css, img, etc.).
var staticFilesHashFS *hashfs.HFS //static files for cache busting using hashfs package.

func init() {
	//Parse flags.
	configFilePath := flag.String("config", "./"+config.DefaultConfigFileName, "Full path to the configuration file.")
	printConfig := flag.Bool("print-config", false, "Print the config file this app has loaded.")
	showVersion := flag.Bool("version", false, "Show the version of the app.")
	showSQLiteVersion := flag.Bool("sqlite-version", false, "Show the version of SQLite the app has embedded.")
	dbDeploySchema := flag.Bool("deploy-db", false, "Deploy a new database or add new tables to an existing database.")
	dbUpdateSchema := flag.Bool("update-db", false, "Update an already deployed database.")
	dbDontInsertInitialData := flag.Bool("no-insert-initial-data", false, "Set to true to deploy the database without inserting default data.") //used when converting from mariadb to sqlite
	logFlags := flag.String("log-prefix", "ymdhms", "Format of logging prefix; none, ymdhms, or ymdhmsmicro.")
	flag.Parse()

	//Handle setting logging prefix. This is useful for handling differences in systems
	//that run the app. In development, it is nice to have prefix timestamp, and
	//sometimes microsecond. However, in production on systems running the app with
	//systemd, systemd/journalctl already prepends the date and time so prefixing the
	//logging output is redundant and makes for longer log lines.
	//
	//This is set via a flag, not config file, because we want the prefix to be set
	//before any logging output when reading and validating the config file.
	switch *logFlags {
	case "none":
		log.SetFlags(0)
	case "ymdhms":
		//default, .SetFlags() does not need to be called.
	case "ymdhmsmicro":
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	default:
		//Catch if something strange was provided, use default prefix. SetFlags() does
		//not need to be called but we do show an error message in case user meant to
		//set log prefix and did so incorrectly.
		log.Println("WARNING! (main) log-prefix flag set to invalid value, using default.")
	}

	//If user just wants to see app version, print it and exit.
	//Not using log.Println() so that a timestamp isn't printed.
	if *showVersion {
		fmt.Println(version.V)
		os.Exit(0)
		return
	}

	//If user just wants to see SQLite version, print it and exit.
	//Not using log.Println() so that a timestamp isn't printed.
	if *showSQLiteVersion {
		ver, err := sqldb.GetSQLiteVersion()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(ver, "-", sqldb.GetSQLiteLibrary())
		os.Exit(0)
		return
	}

	//Starting messages...
	//Always show version number when starting for diagnostics.
	log.Println("Starting License Key Server...")
	log.Printf("Version: %s (Released: %s)\n", version.V, version.ReleaseDate)

	//Read and parse the config file at the provided path. The config file provides
	//runtime configuration of the app and contains settings that are rarely modified.
	// - If the --config flag is blank, the default value, a default config is used.
	// - If the --config flag has a path set, look for a file at the provided path.
	//    - If a file is found, parse it as config file and handle any errors.
	//    - If a file cannot be found, create a default config and save it to the path provided.
	err := config.Read(*configFilePath, *printConfig)
	if err != nil {
		log.Fatalln("Could not handle config file.", err)
		os.Exit(0)
		return
	}

	//Parse source files for building GUI as fs.FS. This allows us handle on-disk or
	//embedded files in the same manner elsewhere (templates, cache busting, root
	//files).
	if config.Data().WebFilesStore == config.WebFilesStoreEmbedded {
		//Get the website/ subdirectory of the embedded files. This is because embed
		//starts at a root "." directory with website being a subdirectory, whereas
		//os.DirFS treats the path/to/website as the root and thus we are "inside"
		//the website/ directory already. This makes handling of embeded versus on-
		//disk files the same.
		//
		//embed:
		// .
		// - website
		//   - static
		//   - templates
		//   - ...
		//
		//on-disk:
		// - static
		// - templates
		// - ...
		sourceFilesFS, err = fs.Sub(embeddedFiles, "website")
		if err != nil {
			log.Fatalln("Could not read website directory", err)
			return
		}
	} else {
		sourceFilesFS = os.DirFS(config.Data().WebFilesPath)
	}

	staticFilesFS, err = fs.Sub(sourceFilesFS, "static")
	if err != nil {
		log.Fatalln("Could not read static directory.", err)
		return
	}

	templateFilesFS, err = fs.Sub(sourceFilesFS, "templates")
	if err != nil {
		log.Fatalln("Could not read templates directory.", err)
		return
	}

	//Handle HTML templates and cache busting.
	staticFilesHashFS = hashfs.NewFS(staticFilesFS)

	pageConfig := pages.Config{
		Development:   config.Data().Development,
		UseLocalFiles: config.Data().UseLocalFiles,
		TemplateFiles: templateFilesFS,
		StaticFiles:   staticFilesHashFS,
	}
	err = pageConfig.ParseTemplates()
	if err != nil {
		log.Fatalln("Could not build templates to build GUI with.", err)
		return
	}

	//Configure the database.
	cfg := &sqldb.Config{
		Type:       sqldb.DBTypeSQLite,
		SQLitePath: config.Data().DBPath,
		SQLitePragmas: []string{
			//Update pages-diagnostics.go to match.
			"PRAGMA busy_timeout = 5000",  //so mattn and modernc are treated the same, 5000 is default for mattn
			"PRAGMA synchronous = NORMAL", //so mattn and modernc are treated the same, NORMAL is default for mattn
			"PRAGMA journal_mode = " + config.Data().DBJournalMode,
			"PRAGMA foreign_keys = ON",
		},
		MapperFunc:    sqldb.DefaultMapperFunc,
		LoggingLevel:  sqldb.LogLevelInfo,
		DeployQueries: db.DeployQueries,
		UpdateQueries: db.UpdateQueries,
		UpdateQueryErrorHandlers: []sqldb.ErrorHandler{
			sqldb.IgnoreErrorDuplicateColumn,
			sqldb.IgnoreErrorDropColumn,
			sqldb.IgnoreErrorDropForeignKey,
			sqldb.IgnoreErrorDropTable,
			sqldb.IgnoreErrorTableDoesNotExist,
			sqldb.IgnoreErrorColumnDoesNotExist,
			sqldb.IgnoreErrorSQLiteModify,
			sqldb.IgnoreErrorRenameDoesNotExist,
			sqldb.IgnoreErrorTableDoesNotExist,
		},
	}
	if !*dbDontInsertInitialData {
		cfg.DeployFuncs = db.DeployFuncs
	}

	sqldb.Use(cfg)

	//Deploy the database if requested by --deploy-db flag.
	if *dbDeploySchema {
		err := sqldb.DeploySchema(nil)
		if err != nil {
			log.Fatalln("Error during db deploy.", err)
			return
		}

		//Log out default user credential is user was created.
		if db.InitialUserPassword != "" {
			log.Println("*********************************************")
			log.Println("Initial User Credentials:")
			log.Println(" Username:", db.InitialUserUsername)
			log.Println(" Password:", db.InitialUserPassword)
			log.Println("*********************************************")

			//Unset the password now that we have logged it out, for security.
			db.InitialUserPassword = ""
		}
	}

	//Update the database if requested by the --update-db flag.
	//
	//Updating does not deploy the db! We did this once but it causes issues when
	//deploying a table index for a table that already exists but the column does not
	//exist (if table is already defined, the CREATE TABLE query doesn't check if
	//every column in the table exists) and the column will be added via an update
	//query. In this case, the CREATE INDEX runs before the ADD COLUMN and thus causes
	//an issue.
	//This "deploy as part of updating" functionality was really done to address adding
	//of new tables to schema. Handle this instead by adding createTable... queries to
	//the list of update queries.
	if *dbUpdateSchema {
		err = sqldb.UpdateSchema(nil)
		if err != nil {
			log.Fatalln("Error during db update.", err)
			return
		}
	}

	//Exit app if user passed the deploy or update flags.
	//
	//This is done so that user doesn't just run the binary with either flag
	//hardcoded (i.e.: in a service file) which could cause issues if the app is
	//updated and restarted (we want users to be very apparent/involved to deploys or
	//updates).
	if *dbDeploySchema || *dbUpdateSchema {
		os.Exit(0)
		return
	}

	//Connect to the database.
	//
	//If the database doesn't exist, it will be created. This "create if doesn't exist"
	//functionality was added to simplify first run of the app (user doesn't have to
	//pass the --deploy-db flag) similar to the creating of a default config file if
	//none exists.
	err = sqldb.Connect()
	if os.IsNotExist(err) {
		log.Println("WARNING! (main) Database file does not exist at given path, file will be created and database will be deployed.")

		err := sqldb.DeploySchema(nil)
		if err != nil {
			log.Fatalln("Error deploying non-existent database.", err)
			return
		}

		//Log out default user credential is user was created.
		if db.InitialUserPassword != "" {
			log.Println("*********************************************")
			log.Println("Initial User Credentials:")
			log.Println(" Username:", db.InitialUserUsername)
			log.Println(" Password:", db.InitialUserPassword)
			log.Println("*********************************************")

			//Unset the password now that we have logged it out, for security.
			db.InitialUserPassword = ""
		}

		//Now that database is created, we can connect to it. The connection used in
		//DeploySchema is closed after deploying is done so we need to reconnect.
		err = sqldb.Connect()
		if err != nil {
			log.Fatalln("Could not connect to db after deploying.", err)
			return
		}

	} else if err != nil {
		log.Fatalln("Could not connect to db.", err)
		return
	}

	//Enable logging of HTTP response errorrs.
	output.Debug(true)
}

func main() {
	defer sqldb.Close()

	//Define middleware.
	secHeaders := alice.New(middleware.SecHeaders)
	auth := secHeaders.Append(middleware.Auth, middleware.LogActivity2)
	admin := auth.Append(middleware.Administrator)
	createLics := auth.Append(middleware.CreateLicenses)
	viewLics := auth.Append(middleware.ViewLicenses)

	//Start the router.
	r := mux.NewRouter()
	r.StrictSlash(true)

	//Handle pages.
	//**login & logout.
	//  Using HandleFunc here instead of Handle with http.HandlerFunc, as below routes,
	//  because we don't need any middlewares here that pass back and forth http.Handlers.
	r.Handle("/", secHeaders.ThenFunc(pages.Login)).Methods("GET")
	r.HandleFunc("/login/", users.Login).Methods("POST")
	r.HandleFunc("/logout/", users.Logout).Methods("GET")

	//**main app pages.
	a := r.PathPrefix("/app").Subrouter()
	a.Handle("/", auth.ThenFunc(pages.Main)).Methods("GET")
	a.Handle("/user-profile/", auth.ThenFunc(pages.UserProfile)).Methods("GET")

	l := a.PathPrefix("/licensing").Subrouter()
	l.Handle("/apps/", admin.ThenFunc(pages.Page)).Methods("GET")
	l.Handle("/licenses/", viewLics.ThenFunc(pages.Page)).Methods("GET")
	l.Handle("/create-license/", createLics.ThenFunc(pages.Page)).Methods("GET")
	l.Handle("/license/", viewLics.ThenFunc(pages.License)).Methods("GET")

	//*admin stuff and settings
	adm := a.PathPrefix("/administration").Subrouter()
	adm.Handle("/users/", admin.ThenFunc(pages.Users)).Methods("GET")
	adm.Handle("/app-settings/", admin.ThenFunc(pages.Page)).Methods("GET")
	adm.Handle("/api-keys/", admin.ThenFunc(pages.Page)).Methods("GET")
	adm.Handle("/user-logins/", admin.ThenFunc(pages.Page)).Methods("GET")
	adm.Handle("/tools/", admin.ThenFunc(pages.Page)).Methods("GET")

	adm.Handle("/activity-log/", admin.ThenFunc(pages.Page)).Methods("GET")
	adm.Handle("/activity-log/activity-over-time-of-day/", admin.ThenFunc(pages.Page)).Methods("GET")
	adm.Handle("/activity-log/max-and-avg-duration-by-month/", admin.ThenFunc(pages.Page)).Methods("GET")
	adm.Handle("/activity-log/duration-of-latest-requests/", admin.ThenFunc(pages.Page)).Methods("GET")
	adm.Handle("/activity-log/duration-by-endpoint/", admin.ThenFunc(pages.Page)).Methods("GET")

	//**diagnostic stuff, accessible without logging in so not on "app" path.
	r.Handle("/diagnostics/", secHeaders.ThenFunc(pages.Diagnostics)).Methods("GET")
	r.HandleFunc("/healthcheck/", healthcheckHandler)

	//**help docs
	help := r.PathPrefix("/help").Subrouter()
	help.Handle("/", http.HandlerFunc(pages.HelpTableOfContents)).Methods("GET")
	help.Handle("/{document}/", http.HandlerFunc(pages.Page)).Methods("GET")

	//
	//
	//API calls (internal to the app, not accesible with api key or outside of app).
	api := r.PathPrefix("/api").Subrouter()

	//**users
	u := api.PathPrefix("/users/").Subrouter()
	u.Handle("/", auth.ThenFunc(users.GetAll)).Methods("GET")
	u.Handle("/add/", admin.ThenFunc(users.Add)).Methods("POST")
	u.Handle("/update/", admin.ThenFunc(users.Update)).Methods("POST")
	u.Handle("/change-password/", admin.ThenFunc(users.ChangePassword)).Methods("POST")
	u.Handle("/2fa/get-qr-code/", admin.ThenFunc(users.Get2FABarcode)).Methods("GET")
	u.Handle("/2fa/verify/", admin.ThenFunc(users.Validate2FACode)).Methods("POST")
	u.Handle("/2fa/deactivate/", admin.ThenFunc(users.Deactivate2FA)).Methods("POST")
	u.Handle("/force-logout/", admin.ThenFunc(users.ForceLogout)).Methods("POST")
	u.Handle("/login-history/clear/", admin.ThenFunc(users.ClearLoginHistory)).Methods("POST")

	u1 := api.PathPrefix("/user").Subrouter()
	u1.Handle("/", auth.ThenFunc(users.GetOne)).Methods("GET")

	//**app settings
	as := api.PathPrefix("/app-settings").Subrouter()
	as.Handle("/", admin.ThenFunc(appsettings.Get)).Methods("GET")
	as.Handle("/update/", admin.ThenFunc(appsettings.Update)).Methods("POST")

	//**api keys
	ak := api.PathPrefix("/api-keys").Subrouter()
	ak.Handle("/", admin.ThenFunc(apikeys.GetAll)).Methods("GET")
	ak.Handle("/generate/", admin.ThenFunc(apikeys.Generate)).Methods("POST")
	ak.Handle("/revoke/", admin.ThenFunc(apikeys.Revoke)).Methods("POST")
	ak.Handle("/update/", admin.ThenFunc(apikeys.Update)).Methods("POST")

	//**activity log
	act := api.PathPrefix("/activity-log").Subrouter()
	act.Handle("/clear/", admin.ThenFunc(activitylog.Clear)).Methods("POST")
	act.Handle("/latest/", admin.ThenFunc(activitylog.GetLatest)).Methods("GET")
	act.Handle("/latest/filter-by-endpoints/", admin.ThenFunc(activitylog.GetLatestEndpoints)).Methods("GET")
	act.Handle("/over-time-of-day/", admin.ThenFunc(activitylog.OverTimeOfDay)).Methods("GET")
	act.Handle("/max-and-avg-monthly-duration/", admin.ThenFunc(activitylog.MaxAndAvgMonthlyDuration)).Methods("GET")
	act.Handle("/latest-requests-duration/", admin.ThenFunc(activitylog.LatestRequestsDuration)).Methods("GET")
	act.Handle("/duration-by-endpoint/", admin.ThenFunc(activitylog.DurationByEndpoint)).Methods("GET")

	//**user logins
	ulg := api.PathPrefix("/user-logins").Subrouter()
	ulg.Handle("/latest/", admin.ThenFunc(users.LatestLogins)).Methods("GET")

	//**apps
	app := api.PathPrefix("/apps").Subrouter()
	app.Handle("/", admin.ThenFunc(apps.Get)).Methods("GET")
	app.Handle("/add/", admin.ThenFunc(apps.Add)).Methods("POST")
	app.Handle("/update/", admin.ThenFunc(apps.Update)).Methods("POST")

	//**keypairs
	kp := api.PathPrefix("/key-pairs").Subrouter()
	kp.Handle("/", admin.ThenFunc(keypairs.Get)).Methods("GET")
	kp.Handle("/add/", admin.ThenFunc(keypairs.Add)).Methods("POST")
	kp.Handle("/delete/", admin.ThenFunc(keypairs.Delete)).Methods("POST")
	kp.Handle("/set-default/", admin.ThenFunc(keypairs.Default)).Methods("POST")

	//**custom fields
	cf := api.PathPrefix("/custom-fields").Subrouter()
	cfd := cf.PathPrefix("/defined").Subrouter()
	cfd.Handle("/", admin.ThenFunc(customfields.GetDefined)).Methods("GET")
	cfd.Handle("/add/", admin.ThenFunc(customfields.Add)).Methods("POST")
	cfd.Handle("/update/", admin.ThenFunc(customfields.Update)).Methods("POST")
	cfd.Handle("/delete/", admin.ThenFunc(customfields.DeleteDefined)).Methods("POST")

	cfr := cf.PathPrefix("/results").Subrouter()
	cfr.Handle("/", viewLics.ThenFunc(customfields.GetResults)).Methods("GET")

	//**licenses
	lics := api.PathPrefix("/licenses").Subrouter()
	lics.Handle("/", viewLics.ThenFunc(license.One)).Queries("id", "").Methods("GET")
	lics.Handle("/", viewLics.ThenFunc(license.All)).Methods("GET")
	lics.Handle("/add/", createLics.ThenFunc(license.Add)).Methods("POST")
	lics.Handle("/download/", viewLics.ThenFunc(license.Download)).Methods("GET")
	lics.Handle("/history/", viewLics.ThenFunc(license.History)).Methods("GET")
	lics.Handle("/notes/", viewLics.ThenFunc(license.Notes)).Methods("GET")
	lics.Handle("/notes/add/", createLics.ThenFunc(license.AddNote)).Methods("POST")
	lics.Handle("/disable/", auth.ThenFunc(license.Disable)).Methods("POST")
	lics.Handle("/renew/", auth.ThenFunc(license.Renew)).Methods("POST")

	//Handle public API endpoints.
	//
	//These are endpoints that are accessible outside of the app using an API key.
	//Only some functionality is available via the public API to prevent any data
	//corruption or other broken functionality.
	//
	//This list of endpoints must match the list defined in middleware-externalAPI.go
	//that checks permissions for the API key being used to access the endpoint.
	externalAPI := alice.New(middleware.ExternalAPI, middleware.LogActivity2)
	extAPI := api.PathPrefix("/v1").Subrouter()
	extAPI.Handle("/licenses/add/", externalAPI.ThenFunc(license.AddViaAPI)).Methods("POST")
	extAPI.Handle("/licenses/download/", externalAPI.ThenFunc(license.Download)).Methods("GET")
	extAPI.Handle("/licenses/renew/", externalAPI.ThenFunc(license.Renew)).Methods("POST")
	extAPI.Handle("/licenses/disable/", externalAPI.ThenFunc(license.Disable)).Methods("POST")

	//Handle static files served off the root directory. This is typically for robots.txt,
	//favicon, etc. {file} is placeholder that isn't used, it is there just so that the
	//router knows to match "something off of /" with this handler.
	r.HandleFunc("/{file}", rootFileHandler)

	//Handle static files/assets. This is anything located of the /static directory
	//and typically includes js, css, images, fonts, etc.
	//
	//See templates.static for more info.
	if config.Data().Development {
		r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFileHeaders(http.FileServer(http.FS(staticFilesFS)))))
	} else {
		r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFileHeaders(hashfs.FileServer(staticFilesHashFS))))
	}

	//Listen and serve.
	port := config.Data().Port
	host := "127.0.0.1"

	hostPort := net.JoinHostPort(host, strconv.Itoa(port))
	log.Println("Listening on port:", port)
	log.Fatal(http.ListenAndServe(hostPort, r))
}

// healthcheckHandler is used to send back a response when an infrastructure
// monitoring tool is checking if this app is running/alive. The sent back
// data could probably be more simple, something like w.Write([]byte("alive")).
func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	output.DataFound("alive", w)
}

// rootFileHandler handles serving static files at the root directory. Think robots.txt
// and favicon.ico.
func rootFileHandler(w http.ResponseWriter, r *http.Request) {
	rootFilesFS, err := fs.Sub(sourceFilesFS, "root")
	if err != nil {
		log.Fatalln("Could not read web root directory.", err)
		return
	}

	http.FileServer(http.FS(rootFilesFS)).ServeHTTP(w, r)
}

// staticFileHeaders sets extra headers when serving static files from our source (not
// CDN source). This is mostly for diagnostics in the browser.
func staticFileHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-web-files-store", config.Data().WebFilesStore)
		next.ServeHTTP(w, r)
	})
}
