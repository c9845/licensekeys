/*
Package pages is used to display the gui. This package is kind of a middleman for
other funcs and the templates package.

This file specifically deals with showing the diagnostics page which shows all
sorts of lower level info on the app.
*/
package pages

import (
	"log"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/version"
	"github.com/c9845/sqldb/v2"
	"github.com/c9845/templates"
)

//startTime is used to track when the app was last started so we can monitor uptime.
//The value for this is set in init() for this package.
var startTime time.Time

func init() {
	startTime = time.Now()
}

//lineKey is used to specify the field name for a line in the diagnostics
type lineKey string

//diagLines is the list of lines we will print in the diagnostics file
//We use this so that we can return the diag data in a consistent order.
type diagLines struct {
	Lines map[lineKey]interface{}
	Order []lineKey
}

//newDiagLines returns an initialized diagLines that new lines can be added to
func newDiagLines() *diagLines {
	var dl diagLines
	dl.Lines = make(map[lineKey]interface{})
	dl.Order = []lineKey{}

	return &dl
}

//set adds a new item to the diagnostics to be printed out.
func (dl *diagLines) set(key string, value interface{}) {
	//get key in correct var type
	//We allow user to provide string as key since it is easier in code.
	k := lineKey(key)

	//save
	dl.Lines[k] = value
	dl.Order = append(dl.Order, k)
}

//Diagnostics shows the diagnostic page.
//We use an actual page for this instead of just using w.Write() so that we
//can display diagnostic info from js and css.
func Diagnostics(w http.ResponseWriter, r *http.Request) {
	//Hold diagnostic data.
	d := newDiagLines()

	//Config file data...
	//Try to maintain same order as config file.
	//Some fields are omitted for security or development.
	cfg := config.Data()
	d.set("**CONFIG**", "******************************")

	d.set("DBPath", cfg.DBPath)
	d.set("DBJournalMode", cfg.DBJournalMode)

	d.set("WebFilesStore", cfg.WebFilesStore)
	d.set("WebFilesPath", cfg.WebFilesPath)
	d.set("StaticFileCacheDays", cfg.StaticFileCacheDays)
	d.set("UseLocalFiles", cfg.UseLocalFiles)
	d.set("FQDN", cfg.FQDN)
	d.set("Port", cfg.Port)

	d.set("LoginLifetimeHours", cfg.LoginLifetimeHours)
	d.set("TwoFactorAuthLifetimeDays", cfg.TwoFactorAuthLifetimeDays)

	//timezone is in TIMEZONE section below
	d.set("MinPasswordLength", cfg.MinPasswordLength)

	//Database diagnostics...
	d.set("**DB Diagnostics**", "******************************")

	stats := sqldb.Connection().Stats()
	d.set("MaxOpenConnections", stats.MaxOpenConnections)
	d.set("Idle", stats.Idle)
	d.set("OpenConnections", stats.OpenConnections)
	d.set("InUse", stats.InUse)
	d.set("WaitCount", stats.WaitCount)
	d.set("WaitDuration", stats.WaitDuration)
	d.set("MaxIdleClosed", stats.MaxIdleClosed)
	d.set("MaxIdleTimeClosed", stats.MaxIdleTimeClosed)
	d.set("MaxLifetimeClosed", stats.MaxLifetimeClosed)

	ver, err := sqldb.GetSQLiteVersion()
	if err != nil {
		log.Println("pages.Diagnostics", "could not get sqlite version", err)
	} else {
		d.set("SQLiteVersion", ver)
	}

	d.set("SQLiteLibrary", sqldb.GetSQLiteLibrary())

	var pragmaJournalMode string
	q := "PRAGMA journal_mode"
	c := sqldb.Connection()
	err = c.Get(&pragmaJournalMode, q)
	if err != nil {
		log.Println("pages.Diagnostics", "could not get sqlite journal mode", err)
	} else {
		d.set("PRAGMAJournalMode", pragmaJournalMode)
	}

	var busyTimeout string
	q = "PRAGMA busy_timeout"
	c = sqldb.Connection()
	err = c.Get(&busyTimeout, q)
	if err != nil {
		log.Println("pages.Diagnostics", "could not get sqlite busy timeout", err)
	} else {
		d.set("PRAGMABusyTimeout", busyTimeout)
	}

	//App settings...
	as, err := db.GetAppSettings(r.Context())
	if err != nil {
		log.Println("diagnostics", "Could not get app settings.", err)
	} else {
		d.set("**APP SETTINGS**", "******************************")

		x := reflect.ValueOf(&as).Elem()
		typeOf := x.Type()
		for i := 0; i < x.NumField(); i++ {
			fieldName := typeOf.Field(i).Name
			value := x.Field(i).Interface()

			d.set(fieldName, value)
		}
	}

	//Misc...
	d.set("**MISC**", "******************************")
	d.set("AppVersion", version.V)
	d.set("ReleaseDate", version.ReleaseDate)
	d.set("OS", runtime.GOOS)
	d.set("Arch", runtime.GOARCH)
	d.set("Startup Time", startTime)

	//adapted from https://www.geeksforgeeks.org/converting-seconds-into-days-hours-minutes-and-seconds/
	diff := int64(time.Since(startTime).Seconds())
	days := int64(diff / (24 * 60 * 60))
	diff = diff % (24 * 60 * 60)
	hours := diff / (60 * 60)
	diff = diff % (60 * 60)
	mins := diff / 60
	diff = diff % 60
	secs := diff
	dhms := strconv.FormatInt(days, 10) + "days, " + strconv.FormatInt(hours, 10) + "hours, " + strconv.FormatInt(mins, 10) + "mins, " + strconv.FormatInt(secs, 10) + "secs"
	d.set("Uptime", dhms)

	//Timezone stuff...
	d.set("**TIMEZONE**", "******************************")
	d.set("Timezone (conf)", cfg.Timezone)
	d.set("Timezone (conf, loaded)", config.GetLocation().String())

	tzName, _ := time.Now().Zone()
	d.set("SytemTimezone (app)", tzName)

	//return data to build page
	pd := PageData{
		Data: *d,
	}

	templates.Show(w, "app", "diagnostics", pd)
}
