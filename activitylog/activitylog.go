/*
Package activitylog handles the logging of user actions performed within the app for
diagnostic and auditing purposes. The activity log records each page view and each
api endpoint hit. The request data is saved but the response data is not (because we
would need to write our own http response writer interface). Activity logging also
works for public api calls.
*/
package activitylog

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/utils"
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
	startDateFromGUI := strings.TrimSpace(r.FormValue("startDate"))
	endDateFromGUI := strings.TrimSpace(r.FormValue("endDate"))

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
	var startDateForQuery, endDateForQuery string
	if startDateFromGUI != "" || endDateFromGUI != "" {
		if startDateFromGUI == "" {
			output.ErrorInputInvalid("You must choose a Start Date since you chose an End Date.", w)
			return
		}
		if endDateFromGUI == "" {
			output.ErrorInputInvalid("You must choose an End Date since you chose a Start Date.", w)
			return
		}

		sd, ed, errMsg, err := utils.ValidateStartAndEndDate(startDateFromGUI, endDateFromGUI, true)
		if err != nil && errMsg != "" {
			output.Error(err, errMsg, w)
			return
		} else if err != nil {
			output.Error(err, "Could not validate the chosen date range.", w)
			return
		} else if errMsg != "" {
			output.ErrorInputInvalid(errMsg, w)
			return
		}

		startDateForQuery = sd
		endDateForQuery = ed
	}

	//Get results.
	activities, err := db.GetActivityLog(r.Context(), userID, apiKeyID, endpoint, searchFor, startDateForQuery, endDateForQuery, uint16(rows))
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
	//Note the "10" in the queries. This defines the time interval/resolution.
	q := `
		SELECT
			COUNT(ID) AS Count,
			
			/**Used for diagnostics.**/
			/* strftime('%H', TIME(DatetimeCreated)) AS Hour_NotModified,*/
			/* strftime('%M', TIME(DatetimeCreated)) AS Minute_NotModified, */

			/**The following two "lines" are used to compensate for the rounding of minutes to 60, which should increment the hours plus one.**/
			/**This compensates for having two rows returned such as "hours: 2, minutes: 60" and "hours: 3, minutes: 0" which are understood to be the same point in time.**/
			(
				CASE
					WHEN (round((strftime('%M', TIME(DatetimeCreated)) * 1.0) / 10) * 10) = 60 THEN strftime('%H', TIME(DatetimeCreated)) + 1
					ELSE strftime('%H', TIME(DatetimeCreated))
				END
			) AS Hour,
			(
				CASE
					WHEN (round((strftime('%M', TIME(DatetimeCreated)) * 1.0) / 10) * 10) = 60 THEN 0
					ELSE (round((strftime('%M', TIME(DatetimeCreated)) * 1.0) / 10) * 10)
				END
			) AS Minute,

			/**This value is used for ordering.**/				
			round(
				(
					(
						(strftime('%H', TIME(DatetimeCreated)) * 60) --Hour
						+ 
						(round((strftime('%M', TIME(DatetimeCreated)) * 1.0) / 10) * 10) --MinuteRounded
					) / 60
				)
			, 2) AS HoursDecimal
			
		FROM ` + db.TableActivityLog + `
		
		GROUP BY HoursDecimal
		ORDER BY HoursDecimal ASC
	`

	//Define custom struct for retreived data since this data is special just for this
	//request.
	type dataReturned struct {
		Count         int //the number of activities that occured within the time interval
		Hour          int
		Minute        int
		MinuteRounded int
		HoursDecimal  float64 //the decimal version of hour:minutes (i.e. 2:15 is 2.25)
	}
	var data []dataReturned

	//Run query.
	c := sqldb.Connection()
	err := c.SelectContext(r.Context(), &data, q)
	if err != nil {
		output.Error(err, "Could not look up data.", w)
		return
	}

	//Allow results to be cached for a short amount of time since this pages takes
	//a while to load (lots of activities) and doesn't change frequently (a few new
	//activities won't drastically change this chart, unless the activity log was
	//recently purged).
	w.Header().Set("Cache-Control", "no-transform,public,max-age="+strconv.Itoa(30))

	output.DataFound(data, w)
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
		GROUP BY strftime("%Y", ` + db.TableActivityLog + `.DatetimeCreated), strftime("%m", ` + db.TableActivityLog + `.DatetimeCreated)`

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
		db.TableActivityLog + `.TimestampCreated`,
		`DATE(` + db.TableActivityLog + `.DatetimeCreated) AS DatetimeCreated`,
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
