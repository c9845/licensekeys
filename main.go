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
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/c9845/cachebusting"
	"github.com/c9845/licensekeys/activitylog"
	"github.com/c9845/licensekeys/apikeys"
	"github.com/c9845/licensekeys/apps"
	"github.com/c9845/licensekeys/appsettings"
	"github.com/c9845/licensekeys/config"
	"github.com/c9845/licensekeys/customfields"
	"github.com/c9845/licensekeys/db"
	"github.com/c9845/licensekeys/keypairs"
	"github.com/c9845/licensekeys/license"
	"github.com/c9845/licensekeys/middleware"
	"github.com/c9845/licensekeys/pages"
	"github.com/c9845/licensekeys/users"
	"github.com/c9845/licensekeys/version"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v2"
	"github.com/c9845/templates"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

//embed HTML templates and static files into app.
//This is done so that we don't need to distribute our HTML, CSS, JS, etc. files
//separately. The end-user will not have to set the WebFilesPath field in their config
//file.
//
//Embedding files will increase the size of the built executable. To limit the size of
//the executable, we only embed the necessary files. This ends up being a lot of files
//since we still want to allow the end-user to serve third party files from the local
//server versus from over the internet/CDN.
//
//go:embed website/templates/*
//go:embed website/static/css
//go:embed website/static/js/*.js
//go:embed website/static/js/vendor
var embeddedFiles embed.FS

func init() {
	//Uncomment this to remove datetimes from logging outputs. May be useful on systemd
	//systems that automatically add in the datetime stamp to logging output in journald.
	// log.SetFlags(0)

	//Parse flags.
	configFilePath := flag.String("config", "./"+config.DefaultConfigFileName, "Full path to the configuration file.")
	printConfig := flag.Bool("print-config", false, "Print the config file this app has loaded.")
	showVersion := flag.Bool("version", false, "Show the version of the app.")
	showSQLiteVersion := flag.Bool("sqlite-version", false, "Show the version of SQLite the app has embedded.")
	dbDeploySchema := flag.Bool("deploy-db", false, "Deploy a new database or add new tables to an existing database.")
	dbUpdateSchema := flag.Bool("update-db", false, "Update an already deployed database.")
	dontInsertInitialData := flag.Bool("no-insert-initial-data", false, "Set to true to deploy the database without inserting default data.") //used when converting from mariadb to sqlite
	flag.Parse()

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
		log.Println("SQLite Version:", ver, err)
		log.Println("SQLite Library:", sqldb.GetSQLiteLibrary())
		os.Exit(0)
		return
	}

	//Starting messages...
	//Always show version number when starting for diagnostics.
	log.Println("Starting License Key Server...")
	log.Println("Version:", version.V)

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

	//Configure the database.
	cfg := sqldb.Config{
		Type:       sqldb.DBTypeSQLite,
		SQLitePath: config.Data().DBPath,
		SQLitePragmas: []string{
			"PRAGMA busy_timeout = 5000", //so mattn and modernc are treated the same, 5000 is default for mattn
			"PRAGMA journal_mode = " + config.Data().DBJournalMode,
		},
		MapperFunc:    sqldb.DefaultMapperFunc,
		DeployQueries: db.DeployQueries,
		DeployFuncs:   db.DeployFuncs,
		UpdateQueries: db.UpdateQueries,
		Debug:         true,
		UpdateIgnoreErrorFuncs: []sqldb.UpdateIgnoreErrorFunc{
			sqldb.UFAddDuplicateColumn,
			sqldb.UFDropUnknownColumn,
			sqldb.UFModifySQLiteColumn,
			sqldb.UFAlreadyExists,
		},
	}
	sqldb.Save(cfg)

	//Deploy the database if requested by --deploy-db flag.
	if *dbDeploySchema {
		err := sqldb.DeploySchema(*dontInsertInitialData)
		if err != nil {
			log.Fatalln("Error during db deploy.", err)
			return
		}
	}

	//Update the database if requested by the --update-db flag.
	//Updating the db also calls the deploy db func since sometimes an app update
	//requires deploying new tables which is done via the deploy functionality, not
	//the update functionality (since we are deploying new tables). This is just
	//done as a tradeoff instead of asking users to pass both the --deploy-db flag
	//and --update-db flag. Yes, the logging for updates is a bit messier but this
	//if acceptable for the ease of use of updating a schema when adding new table(s).
	if *dbUpdateSchema {
		err := sqldb.DeploySchema(*dontInsertInitialData)
		if err != nil {
			log.Fatalln("Error during db deploy during update.", err)
			return
		}

		err = sqldb.UpdateSchema()
		if err != nil {
			log.Fatalln("Error during db update.", err)
			return
		}
	}

	//Exit app if user passed the deploy or update flags. This is done so that user
	//doesn't just run the binary with either flag hardcoded (i.e.: in a service
	//file) which could cause issues if the app is updated and restarted (we want
	//users to be very apparent/involved to deploys or updates).
	if *dbDeploySchema || *dbUpdateSchema {
		os.Exit(0)
		return
	}

	//Connect to the database.
	//If the database doesn't exist, it will be created. This "create if doesn't exist"
	//functionality was added to simplify first run of the app (user doesn't have to
	//pass the --deploy-db flag) similar to the creating of a default config file if
	//none exists.
	err = sqldb.Connect()
	if os.IsNotExist(err) {
		log.Println("WARNING! (main) Database file does not exist at given path, database will be deployed.")

		err := sqldb.DeploySchema(false)
		if err != nil {
			log.Fatalln("Error during db deploy.", err)
			return
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

	//Handle cache busting, creating the cache busting files as needed.
	//This is done before HTML templates are parsed so we can pass cache busting file
	//pairs to the templates for rendering the HTML with the correct static file name.
	if config.Data().WebFilesStore == config.WebFilesStoreOnDisk || config.Data().WebFilesStore == config.WebFilesStoreOnDiskMemory {
		css := cachebusting.NewStaticFile(filepath.Join(config.Data().WebFilesPath, "static", "js", "script.min.js"), path.Join("/", "static", "js", "script.min.js"))
		js := cachebusting.NewStaticFile(filepath.Join(config.Data().WebFilesPath, "static", "css", "styles.min.css"), path.Join("/", "static", "css", "styles.min.css"))
		cachebusting.DefaultOnDiskConfig(css, js)

		if config.Data().WebFilesStore == config.WebFilesStoreOnDiskMemory {
			cachebusting.UseMemory(true)
		}
	} else {
		css := cachebusting.NewStaticFile(filepath.ToSlash(filepath.Join("website", "static", "js", "script.min.js")), path.Join("/", "static", "js", "script.min.js"))
		js := cachebusting.NewStaticFile(filepath.ToSlash(filepath.Join("website", "static", "css", "styles.min.css")), path.Join("/", "static", "css", "styles.min.css"))
		cachebusting.DefaultEmbeddedConfig(embeddedFiles, css, js)
	}
	cachebusting.Development(config.Data().Development)
	err = cachebusting.Create()
	if err == cachebusting.ErrNoCacheBustingInDevelopment {
		log.Println("WARNING! (main) Cache busting is disabled in Development mode!")
	} else if err != nil {
		log.Fatalln("Could not create cache busting files", err)
		return
	}

	//Handles parsing the the HTML templates that display the app UI.
	//This parses and saves the templates for future use.
	if config.Data().WebFilesStore == config.WebFilesStoreOnDisk || config.Data().WebFilesStore == config.WebFilesStoreOnDiskMemory {
		templates.DefaultOnDiskConfig(filepath.Join(config.Data().WebFilesPath, "templates"), []string{"app", "help"})
	} else {
		templates.DefaultEmbeddedConfig(embeddedFiles, "website/templates", []string{"app", "help"})
	}
	templates.Development(config.Data().Development)
	templates.UseLocalFiles(config.Data().UseLocalFiles)
	templates.CacheBustingFilePairs(cachebusting.GetFilenamePairs())
	err = templates.Build()
	if err != nil {
		log.Fatalln("Could not build GUI.", err)
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
	r.Handle("/", secHeaders.Then(http.HandlerFunc(pages.Login))).Methods("GET")
	r.HandleFunc("/login/", users.Login).Methods("POST")
	r.HandleFunc("/logout/", users.Logout).Methods("GET")

	//**main app pages.
	r.Handle("/app/", auth.Then(http.HandlerFunc(pages.Main))).Methods("GET")
	r.Handle("/users/", admin.Then(http.HandlerFunc(pages.Users))).Methods("GET")
	r.Handle("/api-keys/", admin.Then(http.HandlerFunc(pages.App))).Methods("GET")
	r.Handle("/apps/", admin.Then(http.HandlerFunc(pages.App))).Methods("GET")
	r.Handle("/licenses/add/", createLics.Then(http.HandlerFunc(pages.AppMapped))).Methods("GET")
	r.Handle("/licenses/", viewLics.Then(http.HandlerFunc(pages.App))).Methods("GET")
	r.Handle("/license/", viewLics.Then(http.HandlerFunc(pages.License))).Methods("GET")

	//**admin and diagnostic pages.
	r.Handle("/app-settings/", admin.Then(http.HandlerFunc(pages.App))).Methods("GET")
	r.Handle("/user-logins/", admin.Then(http.HandlerFunc(pages.UserLogins))).Methods("GET")
	r.Handle("/activity-log/", admin.Then(http.HandlerFunc(pages.ActivityLog))).Methods("GET")
	r.Handle("/activity-log/charts/activity-over-time-of-day/", admin.Then(http.HandlerFunc(pages.ActivityChartOverTimeOfDay))).Methods("GET")
	r.Handle("/activity-log/charts/max-avg-duration-per-month/", admin.Then(http.HandlerFunc(pages.ActivityChartMaxAvgDuration))).Methods("GET")
	r.Handle("/activity-log/charts/duration-of-latest-requests/", admin.Then(http.HandlerFunc(pages.ActivityChartDurationLatestRequests))).Methods("GET")

	//**misc tools/diagnostics
	r.Handle("/diagnostics/", secHeaders.Then(http.HandlerFunc(pages.Diagnostics))).Methods("GET")
	r.Handle("/tools/", admin.Then(http.HandlerFunc(pages.App))).Methods("GET")

	//**in-app help docs.
	help := r.PathPrefix("/help").Subrouter()
	help.Handle("/", http.HandlerFunc(pages.HelpTableOfContents)).Methods("GET")
	help.Handle("/{document}/", http.HandlerFunc(pages.Help)).Methods("GET")

	//API calls (internal to the app, not accesible with api key or outside of app).
	api := r.PathPrefix("/api").Subrouter()

	//**users
	u := api.PathPrefix("/users/").Subrouter()
	u.Handle("/", auth.Then(http.HandlerFunc(users.GetAll))).Methods("GET")
	u.Handle("/add/", admin.Then(http.HandlerFunc(users.Add))).Methods("POST")
	u.Handle("/update/", admin.Then(http.HandlerFunc(users.Update))).Methods("POST")
	u.Handle("/change-password/", admin.Then(http.HandlerFunc(users.ChangePassword))).Methods("POST")
	u.Handle("/2fa/get-qr-code/", admin.Then(http.HandlerFunc(users.Get2FABarcode))).Methods("GET")
	u.Handle("/2fa/verify/", admin.Then(http.HandlerFunc(users.Validate2FACode))).Methods("POST")
	u.Handle("/2fa/deactivate/", admin.Then(http.HandlerFunc(users.Deactivate2FA))).Methods("POST")
	u.Handle("/force-logout/", admin.Then(http.HandlerFunc(users.ForceLogout))).Methods("POST")

	//**app settings
	as := api.PathPrefix("/app-settings").Subrouter()
	as.Handle("/", admin.Then(http.HandlerFunc(appsettings.Get))).Methods("GET")
	as.Handle("/update/", admin.Then(http.HandlerFunc(appsettings.Update))).Methods("POST")

	//**api keys
	ak := api.PathPrefix("/api-keys").Subrouter()
	ak.Handle("/", admin.Then(http.HandlerFunc(apikeys.GetAll))).Methods("GET")
	ak.Handle("/generate/", admin.Then(http.HandlerFunc(apikeys.Generate))).Methods("POST")
	ak.Handle("/revoke/", admin.Then(http.HandlerFunc(apikeys.Revoke))).Methods("POST")

	//**activity log
	act := api.PathPrefix("/activity-log").Subrouter()
	act.Handle("/clear/", admin.Then(http.HandlerFunc(activitylog.Clear))).Methods("POST")

	//**apps
	app := api.PathPrefix("/apps").Subrouter()
	app.Handle("/", admin.Then(http.HandlerFunc(apps.Get))).Methods("GET")
	app.Handle("/add/", admin.Then(http.HandlerFunc(apps.Add))).Methods("POST")
	app.Handle("/update/", admin.Then(http.HandlerFunc(apps.Update))).Methods("POST")

	//**keypairs
	kp := api.PathPrefix("/key-pairs").Subrouter()
	kp.Handle("/", admin.Then(http.HandlerFunc(keypairs.Get))).Methods("GET")
	kp.Handle("/add/", admin.Then(http.HandlerFunc(keypairs.Add))).Methods("POST")
	kp.Handle("/delete/", admin.Then(http.HandlerFunc(keypairs.Delete))).Methods("POST")
	kp.Handle("/set-default/", admin.Then(http.HandlerFunc(keypairs.Default))).Methods("POST")

	//**custom fields
	cf := api.PathPrefix("/custom-fields").Subrouter()
	cfd := cf.PathPrefix("/defined").Subrouter()
	cfd.Handle("/", admin.Then(http.HandlerFunc(customfields.GetDefined))).Methods("GET")
	cfd.Handle("/add/", admin.Then(http.HandlerFunc(customfields.Add))).Methods("POST")
	cfd.Handle("/update/", admin.Then(http.HandlerFunc(customfields.Update))).Methods("POST")
	cfd.Handle("/delete/", admin.Then(http.HandlerFunc(customfields.DeleteDefined))).Methods("POST")

	cfr := cf.PathPrefix("/results").Subrouter()
	cfr.Handle("/", viewLics.Then(http.HandlerFunc(customfields.GetResults))).Methods("GET")

	//**licenses
	lics := api.PathPrefix("/licenses").Subrouter()
	lics.Handle("/", viewLics.Then(http.HandlerFunc(license.One))).Queries("id", "").Methods("GET")
	lics.Handle("/", viewLics.Then(http.HandlerFunc(license.All))).Methods("GET")
	lics.Handle("/add/", createLics.Then(http.HandlerFunc(license.Add))).Methods("POST")
	lics.Handle("/download/", viewLics.Then(http.HandlerFunc(license.Download))).Methods("GET")
	lics.Handle("/history/", viewLics.Then(http.HandlerFunc(license.History))).Methods("GET")
	lics.Handle("/notes/", viewLics.Then(http.HandlerFunc(license.Notes))).Methods("GET")
	lics.Handle("/notes/add/", createLics.Then(http.HandlerFunc(license.AddNote))).Methods("POST")
	lics.Handle("/disable/", auth.Then(http.HandlerFunc(license.Disable))).Methods("POST")
	lics.Handle("/renew/", auth.Then(http.HandlerFunc(license.Renew))).Methods("POST")

	//Handle public API endpoints.
	//These are endpoints that are accessible outside of the app using an API key.
	//These endpoints are listed here for easy reference for finding what an API key
	//can actually access. Endpoints listed here MUST be listed in the publicEndpoints
	//variable in the apikeys package. This is done to limit what endpoints API keys
	//can actually be used on. For details of what each endpoint is used for, or the
	//data returned, see the apikeys package.
	extAPIMid := alice.New(middleware.ExternalAPI, middleware.LogActivity2)
	externalAPI := api.PathPrefix("/v1").Subrouter()
	externalAPI.Handle("/licenses/add/", extAPIMid.Then(http.HandlerFunc(license.AddViaAPI))).Methods("POST")
	externalAPI.Handle("/licenses/download/", extAPIMid.Then(http.HandlerFunc(license.Download))).Methods("GET")
	externalAPI.Handle("/licenses/renew/", extAPIMid.Then(http.HandlerFunc(license.Renew))).Methods("POST")
	externalAPI.Handle("/licenses/disable/", extAPIMid.Then(http.HandlerFunc(license.Disable))).Methods("POST")

	//Handle static files/assets.
	//This is anything located of the /static directory and typically includes
	//js, css, images, fonts, etc.
	cacheDays := config.Data().StaticFileCacheDays
	if config.Data().Development || config.Data().StaticFileCacheDays < 0 {
		cacheDays = 0
	}
	r.PathPrefix("/static/").Handler(cachebusting.DefaultStaticFileHandler(cacheDays, config.Data().WebFilesPath))

	//Define healthcheck endpoint. This is useful for checking if this app
	//is running using infrastructure monitoring tools. This is allowable on all
	//http methods although GET or HEAD is typical.
	r.HandleFunc("/healthcheck/", healthcheckHandler)

	//Listen and serve.
	//Using host as 127.0.0.1 on macOS (darwin) removes firewall warnings on macOS.
	//Using host as "" is required for app to run in docker container.
	port := config.Data().Port
	host := ""
	if runtime.GOOS == "darwin" {
		host = "127.0.0.1"
	}

	hostPort := net.JoinHostPort(host, strconv.Itoa(port))
	log.Println("Listening on port:", port)
	log.Fatal(http.ListenAndServe(hostPort, r))
}

//healthcheckHandler is used to send back a response when an infrastructure
//monitoring tool is checking if this app is running/alive. The sent back
//data could probably be more simple, something like w.Write([]byte("alive")).
func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	output.DataFound("alive", w)
}
