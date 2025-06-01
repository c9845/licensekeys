package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v4/apikeys"
	"github.com/c9845/licensekeys/v4/db"
	"github.com/c9845/licensekeys/v4/users"
	"golang.org/x/exp/slices"
	"gopkg.in/guregu/null.v3"
)

/*
This file tracks user activity within the app. It saves a record of every page viewed
or endpoint visited so we can investigate user activity as needed.
*/

// skippedEndpoints2 are endpoints we don't need to log to the activity log since they
// would just clog up the log.
var skippedEndpoints2 = []string{}

// LogActivity2 saves the activity the user performed to the database.
//
// This adds a lot of INSERTS to the database which may be undesirable based on server
// load. Therefore, you can disable this via app settings.
//
// Skip on errors since logging isn't the most important thing in the world.
func LogActivity2(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Check if activity logging is enabled in the App Settings.
		as, err := db.GetAppSettings(r.Context())
		if err != nil {
			//Don't error out here since this isn't the end of the world and we want
			//users to still be able to use the app.
			log.Println("middleware.LogActivity2", "could not get app settings to verify if activity logging is enabled, skipped recording activity", err)
			next.ServeHTTP(w, r)
			return
		}
		if !as.EnableActivityLogging {
			next.ServeHTTP(w, r)
			return
		}

		//Skip certain endpoints as needed.
		if slices.Contains(skippedEndpoints2, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		//Skip if user is on the activity log page, don't need to log entries for this
		//page since doing so just clogs us the logging.
		if strings.Contains(r.Referer(), "/activity-log/") {
			next.ServeHTTP(w, r)
			return
		}

		//Start timer to get duration it took for server to response.
		timer := time.Now()

		//Serve the actual page/endpoint.
		next.ServeHTTP(w, r)

		//ServerHTTP will cause the context to be closed, therefore we won't be able
		//to use it below for calls to the db to save data. We have to create a new
		//context for this.
		ctx := context.WithoutCancel(r.Context())

		//Get data from request to save to db for logging purposes. This data is used
		//for diagnostics since the data is the actual data the user provided to this
		//app prior to the app reading or modifying the data in any way. This saves
		//the data as a JSON string so users can copy/paste the data from db or
		//activity log to more easily view in a JSON parser that formats the string
		//nicer. This tries to format the data as JSON and if errors occur falls back
		//to simply flattening the url.Values map[string][]string into a string.
		form := r.Form
		jStrToSave, err := getRequestDataAsJSON2(form)
		if strings.TrimSpace(jStrToSave) == "" || err != nil {
			jStrToSave, err = getRequestDataAsDirtyString2(form)
			if err != nil {
				log.Println("Could not get request data as dirty string.", err)
			}
		}

		//Get IP address. App should be behind a proxy so remote IP is in a header
		//set by proxy.
		ip := r.RemoteAddr
		if v, ok := r.Header["X-Forwarded-For"]; ok {
			ip = strings.Join(v, "")
		}

		referrerFull, err := url.Parse(r.Referer())
		if err != nil {
			referrerFull = &url.URL{}
		}
		referrerPath := referrerFull.Path

		//Data to save.
		activity := db.ActivityLog{
			Method:         r.Method,
			URL:            r.URL.Path,
			RemoteIP:       ip,
			UserAgent:      r.UserAgent(),
			TimeDuration:   time.Since(timer).Nanoseconds() / 1000000, //milliseconds
			PostFormValues: jStrToSave,
			Referrer:       referrerPath, //function name is misspelled, not our field name.
		}

		//Determine if this request was made via an API key or a user and save the
		//respective identifying info.
		//
		//Only one of apiKeyID and userID will be provided.
		//  - apiKeyID is set in middleware.ExternalAPI().
		//  - userID is set in middleware.Auth().
		apiKeyID := ctx.Value(apikeys.APIKeyContextKey)
		userID := ctx.Value(users.UserIDContextKey)

		if apiKeyID != nil {
			activity.CreatedByAPIKeyID = null.IntFrom(apiKeyID.(int64))
			err := activity.Insert(ctx)
			if err != nil {
				log.Println("middleware.LogActivity2", "could not save api access to log", err)
			}
		}

		if userID != nil {
			activity.CreatedByUserID = null.IntFrom(userID.(int64))
			err = activity.Insert(ctx)
			if err != nil {
				log.Println("middleware.LogActivity2", "could not save user access to log", r.URL.Path, err)
			}
		}
	})
}

// getRequestDataAsJSON2 returns the url form values as a JSON string so we don't need
// to use the url.Values map[string][]string format for storing data in db. The
// map[string][]string format is not parsable easily since there are extra characters
// in it. Returning JSON is much more useful for parsing data in a JSON parser/formatter.
//
// Note the use of v[0]. We assume that all requests made to this app will have only
// one value per key. This is done even though the url.Values type is map[string][]string.
// Since we can control how data is provided we can safely assume that there is only
// one value per key at most.
func getRequestDataAsJSON2(formVals url.Values) (output string, err error) {
	//Placeholder for building output.
	jStr2 := make(map[string]interface{})

	//Iterate through each key/value pair, grab the first value, and try
	//converting to a correct format.
	for k, v := range formVals {
		vFirst := v[0]

		//make sure passwords or similar aren't stored in activity log
		if strings.Contains(strings.ToLower(k), "password") {
			vFirst = "****************"
		}
		if strings.Contains(strings.ToLower(k), "twofactorauthsecret") {
			vFirst = "****************"
		}

		if vFirst == "" {
			jStr2[k] = vFirst
			continue
		}

		//integer
		iVal, err := strconv.Atoi(vFirst)
		if err == nil {
			jStr2[k] = iVal
			continue
		}

		//boolean
		bVal, err := strconv.ParseBool(vFirst)
		if err == nil {
			jStr2[k] = bVal
			continue
		}

		//float
		fVal, err := strconv.ParseFloat(vFirst, 64)
		if err == nil {
			jStr2[k] = fVal
			continue
		}

		//null
		if vFirst == "null" {
			jStr2[k] = nil
		}

		//object
		var vObject map[string]interface{}
		err = json.Unmarshal([]byte(vFirst), &vObject)
		if err == nil {
			jStr2[k] = vObject
			continue
		}

		//array (of objects must likely)
		var vArray []map[string]interface{}
		err = json.Unmarshal([]byte(vFirst), &vArray)
		if err == nil {
			jStr2[k] = vArray
			continue
		}

		//default to string
		jStr2[k] = vFirst
	}

	//Get map[string]interface as json string.
	j, err := json.Marshal(jStr2)
	if err != nil {
		log.Println("Could not marshal into json.", err)
		return
	}
	output = string(j)

	return
}

// getRequestDataAsDirtyString2 returns the url form values as a map[string][]string
// flattened using a JSON format. The data in it is not correctly formatted as JSON
// and cannot be parsed as JSON. This is used as a backup to getRequestDataAsJSON2
// failing to parse correctly so that we still record a copy of the request data. The
// data can be manually formatted as JSON if needed.
func getRequestDataAsDirtyString2(formVals url.Values) (output string, err error) {
	j, err := json.Marshal(formVals)
	if err != nil {
		return
	}

	output = strings.ReplaceAll(string(j), "\\", "")
	output = strings.ReplaceAll(output, ",", ", ")
	return
}
