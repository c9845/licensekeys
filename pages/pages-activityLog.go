/*
Package pages is used to display the gui. This package is kind of a middleman for
other funcs and the templates package.

This file specifically deals with showing the activity log.
*/
package pages

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/sqldb/v2"
	"github.com/c9845/templates"
)

//ActivityLog shows the page of user activity within the app. This is useful for
//auditing user activity or for diagnnostics (seeing what data a user provided versus
//what they say when an error occurs). You can filter the results by a specific user
//and/or by a search string. Using a search string performs a sql LIKE query with
//starting & ending wildcards on the request data (post form values).
func ActivityLog(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)
	endpoint := strings.TrimSpace(r.FormValue("url"))
	searchFor := strings.TrimSpace(r.FormValue("searchFor"))
	rows, _ := strconv.ParseInt(r.FormValue("rows"), 10, 64)

	//Validate.
	if userID < 0 {
		userID = 0
	}
	if rows < 0 {
		rows = 200
	}

	//Get results.
	activities, err := db.GetActivityLog(r.Context(), userID, endpoint, searchFor, uint16(rows))
	if err != nil {
		e := ErrorPage{
			PageTitle:   "View Activity Log",
			Topic:       "An error occured while trying to display the activity log.",
			Message:     "The activity log data cannot be displayed.",
			Solution:    "Please contact an administrator and have them look at the logs to investigate.",
			ShowLinkBtn: true,
		}
		log.Println("pages.ActivityLog", "Could not look up list of activities", err)

		ShowError(w, r, e)
		return
	}

	//Look up list of endpoints so user can filter by endpoint. This looks up the
	//endpoints for the last 90 days just to reduce the time it takes to run this
	//query. The data will be used to build a <select> menu which will set a url
	//param and refresh the page.
	c := sqldb.Connection()
	q := `
		SELECT URL
		FROM ` + db.TableActivityLog + ` 
		WHERE DatetimeCreated > ?
		GROUP BY URL
		ORDER BY URL
	`
	const lastXDays = 90
	since := time.Now().AddDate(0, 0, -lastXDays).Format("2006-01-02")
	var urls []string
	err = c.Select(&urls, q, since)
	if err != nil {
		log.Println("pages.ActivityLog", "could not look up endpoints to filter by", err)
		//not exiting on error since we can still generate page with list of activities
	}

	//Get list of users to filter by. This allows an end user to view a specific
	//user's activities. The data will be used to build a <select> menu which will
	//set a url param and refresh the page.
	users, err := db.GetUsers(r.Context(), true)
	if err != nil {
		log.Println("pages.ActivityLog", "could not get list of users to filter by", err)
		//not exiting on error since we can still generate page with list of activities
	}

	//Get data to build gui.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("pages.ActivityLog", "Error getting page config data", err)
		return
	}

	//Data to build page with.
	data := struct {
		Activities []db.ActivityLog //list of activities to display
		URLs       []string         //url endpoints to filter by
		Users      []db.User        //users to filter by

		//values chosen by users, so we can populate inputs/selects with user provided values
		//so that when page refreshes the user sees the options they chose.
		UserChosen     int64
		EndpointChosen string
		RowCount       int64
		SearchFor      string
	}{
		Activities: activities,
		URLs:       urls,
		Users:      users,

		UserChosen:     userID,
		EndpointChosen: endpoint,
		RowCount:       rows,
		SearchFor:      searchFor,
	}

	//Show page.
	pd.Data = data
	templates.Show(w, "app", "activity-log", pd)
}

//ActivityChartOverTimeOfDay shows the usage of the app over the time of day broken
//into 10 minute increments. This is useful for showing when users are most active in
//the app. This query/page load will look up a lot of data and take a lot of time to
//complete.
func ActivityChartOverTimeOfDay(w http.ResponseWriter, r *http.Request) {
	//This query is very specialized for this page so there is no need to store it in
	//the db package.
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

	//Define custom struct for retreived data since this data is special just for this page.
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
		e := ErrorPage{
			PageTitle:   "View Activity Chart",
			Topic:       "An error occured while trying to retrieve the data to build the chart.",
			Solution:    "Please contact an administrator and have them look at the logs to investigate.",
			ShowLinkBtn: true,
		}
		log.Println("pages.ActivityLogChart", "Could not look up data", err)

		ShowError(w, r, e)
		return
	}

	//Get data as json. This will be embedded in hidden input element which will be
	//read by Vue/JS and used to build chart. Kind of a hacky way to do this but it
	//works!
	j, _ := json.Marshal(data)

	//Get data to build gui.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("pages.ActivityChartOverTimeOfDay", "Error getting page config data", err)
		return
	}

	//Show page.
	pd.Data = string(j)
	templates.Show(w, "app", "activity-log-chart-over-time-of-day", pd)
}

//ActivityChartMaxAvgDuration shows the average and max duration of requests made to
//the app on a monthly basis. This can be used to check if "things are getting slower".
func ActivityChartMaxAvgDuration(w http.ResponseWriter, r *http.Request) {
	//Handle options, mostly for advanced usage.
	ignorePDF, _ := strconv.ParseBool(r.FormValue("ignorePDF")) //ignores looking up of pdf files since if we have to create the pdf file this can create a lot of latency and push the avg, or max, way off typical

	//Always return data as FLOOR()/ROUND() since this is nicer to display in the GUI (ints vs float/decimal).
	cols := sqldb.Columns{
		`ROUND(AVG(` + db.TableActivityLog + `.TimeDuration)) AS AverageTimeDuration`,
		`ROUND(MAX(` + db.TableActivityLog + `.TimeDuration)) AS MaxTimeDuration`,
		`strftime("%Y", ` + db.TableActivityLog + `.DatetimeCreated) AS Year`,
		`strftime("%m", ` + db.TableActivityLog + `.DatetimeCreated) AS Month`,
	}
	groupBy := `strftime("%Y", ` + db.TableActivityLog + `.DatetimeCreated),	strftime("%m", ` + db.TableActivityLog + `.DatetimeCreated)`

	//Determine if we need to do any filtering.
	wheres := []string{}
	if ignorePDF {
		w := `(` + db.TableActivityLog + `.PostFormValues NOT LIKE '%"type":"pdf"%')`
		wheres = append(wheres, w)
		log.Println("pages.ActivityChartMaxAvgDuration", "ignoring type:pdf endpoints")
	}

	//Build query.
	colString, err := cols.ForSelect()
	if err != nil {
		e := ErrorPage{
			PageTitle:   "View Activity Log Duration",
			Topic:       "An error occured while trying to retrieve the data to build the chart.",
			Solution:    "Please contact an administrator and have them look at the logs to investigate.",
			ShowLinkBtn: true,
		}
		log.Println("pages.ActivityChartMaxAvgDuration", "Could not build column string", err)

		ShowError(w, r, e)
		return
	}

	q := `SELECT ` + colString + ` FROM ` + db.TableActivityLog

	if len(wheres) > 0 {
		q += ` WHERE ` + strings.Join(wheres, " AND ")
	}

	q += ` GROUP BY ` + groupBy

	//Define custom struct to return data with.
	type dataReturned struct {
		AverageTimeDuration int
		MaxTimeDuration     int
		Year                string
		Month               string
	}
	var data []dataReturned

	//Run query.
	c := sqldb.Connection()
	err = c.SelectContext(r.Context(), &data, q)
	if err != nil {
		e := ErrorPage{
			PageTitle:   "View Activity Log Duration",
			Topic:       "An error occured while trying to retrieve the data to build the chart.",
			Solution:    "Please contact an administrator and have them look at the logs to investigate.",
			ShowLinkBtn: true,
		}
		log.Println("pages.ActivityChartMaxAvgDuration", "Could not look up data", err)

		ShowError(w, r, e)
		return
	}

	//Get data as json. This will be embedded in hidden input element which will be
	//read by Vue/JS and used to build chart. Kind of a hacky way to do this but it
	//works!
	j, _ := json.Marshal(data)

	//Get data to build gui.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("pages.ActivityChartMaxAvgDuration", "Error getting page config data", err)
		return
	}

	//Show page.
	pd.Data = string(j)
	templates.Show(w, "app", "activity-log-chart-max-avg-duration", pd)
}

//ActivityChartDurationLatestRequests shows duration of the latest requests to the app.
//This defaults to the latest 500 requests but this can be changed as needed, but note
//that the time to serve the data may get longer.
func ActivityChartDurationLatestRequests(w http.ResponseWriter, r *http.Request) {
	//Handle options, mostly for advanced usage.
	limit, _ := strconv.Atoi(r.FormValue("limit"))

	//Columns.
	cols := sqldb.Columns{
		db.TableActivityLog + `.Method`,
		db.TableActivityLog + `.URL`,
		db.TableActivityLog + `.TimeDuration`,
		db.TableActivityLog + `.TimestampCreated`,
		`DATE(` + db.TableActivityLog + `.DatetimeCreated) AS DatetimeCreated`,
	}

	colstring, err := cols.ForSelect()
	if err != nil {
		e := ErrorPage{
			PageTitle:   "View Activity Log Duration",
			Topic:       "An error occured while trying to retrieve the data to build the chart.",
			Solution:    "Please contact an administrator and have them look at the logs to investigate.",
			ShowLinkBtn: true,
		}
		log.Println("pages.ActivityChartDurationLatestRequests", "Could not build column string", err)

		ShowError(w, r, e)
		return
	}

	//Build query.
	q := `
		SELECT ` + colstring + `
		FROM ` + db.TableActivityLog + ` 
		ORDER BY ` + db.TableActivityLog + `.TimestampCreated DESC`

	if limit <= 0 {
		q += " LIMIT 500"
	}

	//Wrap query in another SELECT to reverse order (so chart can be printed oldest on
	//left). Need to do this since to get latest results we need to ORDER BY...DESC
	//but we want to show results in ASC.
	q2 := `WITH to_be_reversed AS ( ` + q + `) SELECT * FROM to_be_reversed ORDER BY TimestampCreated ASC`

	//Run query.
	var result []db.ActivityLog
	c := sqldb.Connection()
	err = c.SelectContext(r.Context(), &result, q2)
	if err != nil {
		e := ErrorPage{
			PageTitle:   "View Activity Log Duration",
			Topic:       "An error occured while trying to retrieve the data to build the chart.",
			Solution:    "Please contact an administrator and have them look at the logs to investigate.",
			ShowLinkBtn: true,
		}
		log.Println("pages.ActivityChartDurationLatestRequests", "could not get data", err)

		ShowError(w, r, e)
		return
	}

	//Get data as json. This will be embedded in hidden input element which will be
	//read by Vue/JS and used to build chart. Kind of a hacky way to do this but it
	//works!
	j, _ := json.Marshal(result)

	//Get data to build gui.
	pd, err := getPageConfigData(r)
	if err != nil {
		log.Println("pages.ActivityChartDurationLatestRequests", "Error getting page config data", err)
		return
	}

	//Show page.
	pd.Data = string(j)
	templates.Show(w, "app", "activity-log-chart-duration-latest-requests", pd)
}
