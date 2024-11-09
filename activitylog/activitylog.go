/*
Package activitylog handles the logging of user actions performed within the app for
diagnostic and auditing purposes. The activity log records each page view and each
api endpoint hit. The request data is saved but the response data is not (because we
would need to write our own http response writer interface). Activity logging also
works for public api calls.
*/
package activitylog

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v3/config"
	"github.com/c9845/licensekeys/v3/db"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
)

// Clear handles deleting rows from the activity log table. This is only done from the
// admin tools page and is done to clean up the database since the activity log table
// can get very big.
//
// The user provides a starting date to delete from, this way you can delete very old
// activity log rows but keep newer history.
func Clear(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	priorToDate := strings.TrimSpace(r.FormValue("priorToDate"))

	//Validate.
	if len(priorToDate) != len("2006-02-02") {
		output.ErrorInputInvalid("Invalid date provided. Date must be in YYYY-MM-DD format and should be a date in the past.", w)
		return
	}

	//Delete.
	rowsDeleted, err := db.ClearActivityLog(r.Context(), priorToDate)
	if err != nil {
		output.Error(err, "Could not clear activity log.", w)
		return
	}

	output.UpdateOKWithData(rowsDeleted, w)
}

// GetLatest retrieves the data of the latest user and API actions within the app. This
// is useful for auditing user activity and for diagnnostics (seeing what data a user
// provided versus what they say when an error occurs). You can filter the results by
// a specific user, a specific API key, a specific endpoint, and/or a search string.
// Using a search string performs a sql LIKE query with starting & ending wildcards
// on the request data (post form values).
func GetLatest(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)
	apiKeyID, _ := strconv.ParseInt(r.FormValue("apiKeyID"), 10, 64)
	endpoint := strings.TrimSpace(r.FormValue("endpoint"))
	searchFor := strings.TrimSpace(r.FormValue("searchFor"))
	rows, _ := strconv.ParseInt(r.FormValue("rows"), 10, 64)
	startDate := strings.TrimSpace(r.FormValue("startDate"))
	endDate := strings.TrimSpace(r.FormValue("endDate"))

	//Validate.
	if userID < 0 {
		//No error, just use default "don't filter by user".
		userID = 0
	}
	if apiKeyID < 0 {
		//No error, just use default "don't filter by api key".
		apiKeyID = 0
	}
	if userID > 0 && apiKeyID > 0 {
		output.ErrorInputInvalid("Please choose a User or an API Key, not both.", w)
		return
	}
	if rows < 0 {
		//No error, just use default. Limit rows returned to be faster.
		rows = 200
	}

	//Validate date range, if provided.
	//
	//Date range is optional. If it isn't provided, the most recent results will be
	//returned.
	if startDate != "" && endDate != "" {
		startDateParsed, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			output.Error(err, "Could not parse start date.", w)
			return
		}
		endDateParsed, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			output.Error(err, "Could not parse end date.", w)
			return
		}
		if startDateParsed.After(endDateParsed) {
			output.ErrorInputInvalid("Start date must be before end date.", w)
			return
		}
	}

	//Get results.
	activities, err := db.GetActivityLog(r.Context(), userID, apiKeyID, endpoint, searchFor, startDate, endDate, uint16(rows))
	if err != nil {
		output.Error(err, "Could not get latest activities.", w)
		return
	}

	//Allow results to be cached for a short amount of time since this pages takes
	//a while to load (lots of activities).
	const cacheSeconds = 30
	w.Header().Set("Cache-Control", "no-transform,public,max-age="+strconv.Itoa(cacheSeconds))

	output.DataFound(activities, w)
}

// GetLatestEndpoints looks up the list of a user can pick from to filter the list of
// latest activites by (see GetLatest). This simply looks up the list of endpoints
// "hit" over the last 30 days.
func GetLatestEndpoints(w http.ResponseWriter, r *http.Request) {
	//Build query.
	c := sqldb.Connection()
	q := `
		SELECT URL
		FROM ` + db.TableActivityLog + ` 
		WHERE DatetimeCreated > ?
		GROUP BY URL
		ORDER BY URL
	`

	//Get "date since" to look up endpoints used.
	const lastXDays = 60 //Anything longer than 60 days made this query take 2+ secs to return to the browser.
	since := time.Now().AddDate(0, 0, -lastXDays).Format("2006-01-02")
	var endpoints []string

	//Run query.
	err := c.SelectContext(r.Context(), &endpoints, q, since)
	if err != nil {
		output.Error(err, "Could not look up latest endpoints used.", w)
		return
	}

	output.DataFound(endpoints, w)
}

// OverTimeOfDay gets the number of requests per time of day. This is used to tell you
// when the app is most used (when your users are most active in the app).
func OverTimeOfDay(w http.ResponseWriter, r *http.Request) {
	//Get query.
	//
	//Time bracket divides hours into 10 minute intervals. Using 10 minute intervals,
	//instead of per-minute intervals, just makes viewing the charted data easier.
	//
	//Note WITH and secondary query. This was done so that we don't have to rewrite
	//the queries to get _MinutesRaw, _MinutesRounded, and _HoursRaw over and over,
	//instead we can just reference them in CASE statements. This is really all about
	//reducing retyping and possibly typoing something in a retype.
	//
	//Note slight differences in queries, especially around CAST statements.
	const timeBracketSizeMinutes = "10" //string for easier concatting in query.
	offset := config.GetTimezoneOffsetForSQLite()
	q := `
		WITH t AS (
			SELECT
				/* Activity/row. */
				ID,
				
				/* Time bracket minutes. */
				/* Note math to calculate bracket: 
					1) Multiply by 1.0 to get a decimal so we can do decimal division (46 -> 46.0).
					2) Divide by 10 to get number as a decimal we can round up/down to whole number.
					3) Multiply by 10 to get number back to a bracket of 10 minute intervals.
				*/
				(strftime("%M", datetime(DatetimeCreated, '` + offset + `'))) AS MinutesRaw,
				(
					ROUND(
						((strftime(
							"%M",
							datetime(
								DatetimeCreated, 
								'` + offset + `'
							)
						) * 1.0) / ` + timeBracketSizeMinutes + `)
					)* 10
				) AS MinutesRounded,
				
				/* Get hour bracket. */
				(strftime("%H", datetime(DatetimeCreated, '` + offset + `'))) AS HourRaw
				
			FROM activity_log
		)
		
		SELECT
			/* Count number of activities per time bracket. */
			COUNT(ID) AS Count,
			
			/* Raw data for diagnostics. */
			MinutesRaw,
			CAST(MinutesRounded AS INT) AS MinutesRounded,
			HourRaw,
			
			/* Handle minute bracket round-up to 60, which is the same as the 0 minute bracket for next hour. */
			/* AKA "hour 2, minutes 60" is the same as "hour 3, minute 0". */
			(CASE
				WHEN MinutesRaw = 60 THEN HourRaw + 1
				ELSE HourRaw
			END) AS HourBracket,
			(CASE
				WHEN MinutesRounded = 60 THEN 0
				ELSE MinutesRounded
			END) AS MinuteBracket,
			
			/* For ordering. */
			ROUND(HourRaw + (MinutesRounded / 60), 2) AS HoursMinutesDecimal
			
		FROM t
		GROUP BY HoursMinutesDecimal
		ORDER BY HoursMinutesDecimal ASC
	`

	//Define custom struct for retreived data since this data is special just for this
	//function.
	type row struct {
		//The number of activities that occurred within the time bracket/interval.
		Count int

		//Raw data for diagnostics.
		MinutesRaw     int //00 - 60
		MinutesRounded int //00, 10, 20, 30 40, 50, 60
		HourRaw        int //00 - 24

		//Calculated time brackets, taking into consideration minute bracket overlap.
		//AKA hour 2 minute 60 is the same as hour 3 minute 0.
		HourBracket   int
		MinuteBracket int

		//Time as a decimal for ordering.
		HoursMinutesDecimal float64 //3:30 = 3.5
	}
	reportData := []row{}

	//Run query.
	c := sqldb.Connection()
	err := c.SelectContext(r.Context(), &reportData, q)
	if err != nil {
		output.Error(err, "Could not look up data.", w)
		return
	}

	//Allow results to be cached for a short amount of time since this pages takes
	//a while to load (lots of activities) and doesn't change frequently (a few new
	//activities won't drastically change this chart, unless the activity log was
	//recently purged).
	w.Header().Set("Cache-Control", "no-transform,public,max-age="+strconv.Itoa(30))

	output.DataFound(reportData, w)
}

// MaxAndAvgMonthlyDuration retrieves the maximum and average duration of times it
// took the app/server to respond to an HTTP request. This measures app/server latency
// in handling requests and is useful to see if latency is increasing for some reason.
func MaxAndAvgMonthlyDuration(w http.ResponseWriter, r *http.Request) {
	//Get query.
	//
	//Always return data as FLOOR()/ROUND() since this is nicer to display in the GUI
	//(ints vs float/decimal).
	//
	//DATE_FORMAT is used over MONTH so that we can get two character month codes for
	//Jan through Sep (beginning zero).
	cols := sqldb.Columns{
		`ROUND(AVG(` + db.TableActivityLog + `.TimeDuration)) AS AverageTimeDuration`,
		`ROUND(MAX(` + db.TableActivityLog + `.TimeDuration)) AS MaxTimeDuration`,
		`strftime("%Y", ` + db.TableActivityLog + `.DatetimeCreated) AS Year`,
		`strftime("%m", ` + db.TableActivityLog + `.DatetimeCreated) AS Month`,
	}

	colString, err := cols.ForSelect()
	if err != nil {
		output.Error(err, "Could not build columns to select", w)
		return
	}

	//Build query.
	q := `
		SELECT ` + colString + ` 
		FROM ` + db.TableActivityLog + `
		GROUP BY strftime("%Y-%m", ` + db.TableActivityLog + `.DatetimeCreated)`

	//Define custom struct for retreived data since this data is special just for this
	//request.
	type dataReturned struct {
		AverageTimeDuration int
		MaxTimeDuration     int
		Year                string
		Month               string
	}
	var data []dataReturned

	//Run query
	c := sqldb.Connection()
	err = c.SelectContext(r.Context(), &data, q)
	if err != nil {
		output.Error(err, "Could not look up data.", w)
		return
	}

	//Allow results to be cached for a short amount of time since this pages takes
	//a while to load (lots of activities) and doesn't change frequently.
	w.Header().Set("Cache-Control", "no-transform,public,max-age="+strconv.Itoa(30))

	output.DataFound(data, w)
}

// LatestRequestsDuration gets the duration it took the app/server to respond to the
// latest HTTP requests.
func LatestRequestsDuration(w http.ResponseWriter, r *http.Request) {
	//Handle options, mostly for advanced usage.
	limit, _ := strconv.Atoi(r.FormValue("limit")) //don't return too many results b/c then chart is unreadable.

	//Get query.
	cols := sqldb.Columns{
		db.TableActivityLog + `.Method`,
		db.TableActivityLog + `.URL`,
		db.TableActivityLog + `.TimeDuration`,
		`DATE(` + db.TableActivityLog + `.DatetimeCreated) AS DatetimeCreated`,
		db.TableActivityLog + `.TimestampCreated`,
	}

	colstring, err := cols.ForSelect()
	if err != nil {
		output.Error(err, "Could not build columns to select", w)
		return
	}

	//Build query.
	q := `
		SELECT ` + colstring + ` 
		FROM ` + db.TableActivityLog + ` 
		ORDER BY ` + db.TableActivityLog + `.TimestampCreated DESC
	`

	if limit <= 0 {
		q += " LIMIT 250"
	} else {
		q += " LIMIT " + strconv.Itoa(limit)
	}

	//Wrap query in another SELECT to reverse order so chart can be printed oldest
	//on left, like a timeline. Need to do this since to get latest results we need
	//to use ORDER BY...DESC but we want to show results in ASC.
	q2 := `WITH to_be_reversed AS ( ` + q + `) SELECT * FROM to_be_reversed ORDER BY TimestampCreated ASC`

	//Run query.
	var data []db.ActivityLog
	c := sqldb.Connection()
	err = c.SelectContext(r.Context(), &data, q2)
	if err != nil {
		output.Error(err, "Could not look up data.", w)
		return
	}

	//Allow results to be cached for a short amount of time since this pages takes
	//a while to load (lots of activities) and doesn't change frequently.
	w.Header().Set("Cache-Control", "no-transform,public,max-age="+strconv.Itoa(30))

	output.DataFound(data, w)
}

// DurationByEndpoint gets the duration it took the server/app the response grouped
// by endpoint. This is useful for identifying the slowest/most latent endpoints.
func DurationByEndpoint(w http.ResponseWriter, r *http.Request) {
	//Get query.
	//
	//Ignore DatetimeCreated timezone handling here since this data is pretty low-
	//level and doesn't need to be adjusted.
	cols := sqldb.Columns{
		db.TableActivityLog + `.URL`, //endpoint
		`COUNT(ID) AS EndpointHits`,  //for displaying how popular the endpoint is, infrequently hit endpoints that are slow aren't an issue.
		`AVG(` + db.TableActivityLog + `.TimeDuration) AS AverageTimeDuration`,
		`MAX(` + db.TableActivityLog + `.TimeDuration) AS MaxTimeDuration`,
		`MIN(` + db.TableActivityLog + `.TimeDuration) AS MinTimeDuration`,
	}

	colstring, err := cols.ForSelect()
	if err != nil {
		output.Error(err, "Could not build columns to select", w)
		return
	}

	//Build query.
	q := `
		SELECT ` + colstring + ` 
		FROM ` + db.TableActivityLog

	q += ` GROUP BY ` + db.TableActivityLog + `.URL`
	q += ` ORDER BY MAX(` + db.TableActivityLog + `.TimeDuration) DESC`

	//Define custom struct for retreived data since this data is special just for this
	//request.
	type dataReturned struct {
		URL                 string
		EndpointHits        int
		Method              string
		AverageTimeDuration float64
		MaxTimeDuration     float64
		MinTimeDuration     float64
	}
	var data []dataReturned

	//Run query.
	c := sqldb.Connection()
	err = c.SelectContext(r.Context(), &data, q)
	if err != nil {
		output.Error(err, "Could not look up data.", w)
		return
	}

	//Round times to whole numbers for cleaner displaying. This isn't done with ROUND()
	//in SQL query because SQLite doesn't have ROUND() unless it is built with it.
	//This is possible for mattn, but not for modernc.
	for k, v := range data {
		data[k].AverageTimeDuration = math.Floor(v.AverageTimeDuration)
		data[k].MaxTimeDuration = math.Floor(v.MaxTimeDuration)
		data[k].MinTimeDuration = math.Floor(v.MinTimeDuration)
	}

	//Allow results to be cached for a short amount of time since this pages takes
	//a while to load (lots of activities) and doesn't change frequently.
	w.Header().Set("Cache-Control", "no-transform,public,max-age="+strconv.Itoa(60))

	output.DataFound(data, w)
}
