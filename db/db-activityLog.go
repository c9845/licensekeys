package db

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/timestamps"
	"github.com/c9845/sqldb/v3"
	"gopkg.in/guregu/null.v3"
)

//This table stores a history of user and API interactions with this app. This is meant
//for use for auditing purposes.

// TableActivityLog is the name of the table
const TableActivityLog = "activity_log"

// ActivityLog is used to interact with the table
type ActivityLog struct {
	ID               int64
	DatetimeCreated  string //no default in CREATE TABLE so that we can set value using golang since we will also set TimestampCreated using golang.
	TimestampCreated int64  //nanoseconds, for sorting, no default in CREATE TABLE since handling nanoseconds with DEFAULT and SQLite is a pain/impossible.
	Method           string //GET, POST, etc.
	URL              string //the endpoint accessed
	RemoteIP         string
	UserAgent        string
	TimeDuration     int64  //milliseconds it took for server to complete the request
	PostFormValues   string //json encoded form values passed in request
	Referrer         string //page the user is on that caused this request to be made.

	//either CreatedByUserID or CreatedByAPIKeyID is provided, never both
	CreatedByUserID   null.Int
	CreatedByAPIKeyID null.Int

	//JOINed fields
	Username          string
	APIKeyDescription string
	APIKeyK           string //the actual api key

	//Calculated fields
	DatetimeCreatedInTZ string //DatetimeCreated converted to timezone per config file.
}

const (
	createTableActivityLog = `
		CREATE TABLE IF NOT EXISTS ` + TableActivityLog + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT NOT NULL,
			TimestampCreated INTEGER NOT NULL,
			Method TEXT NOT NULL,
			URL TEXT NOT NULL,
			RemoteIP TEXT NOT NULL,
			UserAgent TEXT NOT NULL,
			TimeDuration INTEGER NOT NULL DEFAULT 0,
			PostFormValues TEXT NOT NULL DEFAULT '',
			Referrer TEXT NOT NULL DEFAULT "",
			
			CreatedByUserID INTEGER DEFAULT NULL,
			CreatedByAPIKeyID INTEGER DEFAULT NULL,
			
			FOREIGN KEY(CreatedByUserID) REFERENCES ` + TableUsers + `(ID),
			FOREIGN KEY(CreatedByAPIKeyID) REFERENCES ` + TableAPIKeys + `(ID)
		)
	`

	//indexes
	createIndexActivityLogTimestampCreated = `CREATE INDEX IF NOT EXISTS ` + TableActivityLog + `__TimestampCreated_idx ON ` + TableActivityLog + ` (TimestampCreated)`
	createIndexActivityLogDatetimeCreated  = `CREATE INDEX IF NOT EXISTS ` + TableActivityLog + `__DatetimeCreated_idx ON ` + TableActivityLog + ` (DatetimeCreated)`

	//updates
)

// Insert saves a log entry to the database for an action performed by a user or via
// an API key.
func (a *ActivityLog) Insert(ctx context.Context) (err error) {
	//Common fields between user and API actions.
	cols := sqldb.Columns{
		"Method",
		"URL",
		"RemoteIP",
		"UserAgent",
		"TimeDuration",
		"PostFormValues",
		"DatetimeCreated",
		"TimestampCreated",
		"Referrer",
	}
	b := sqldb.Bindvars{
		a.Method,
		a.URL,
		a.RemoteIP,
		a.UserAgent,
		a.TimeDuration,
		a.PostFormValues,
		timestamps.YMDHMS(),
		time.Now().UnixNano(),
		a.Referrer,
	}

	//Add fields based on if this action was caused by user or API.
	if a.CreatedByUserID.Int64 > 0 {
		cols = append(cols, "CreatedByUserID")
		b = append(b, a.CreatedByUserID.Int64)
	} else if a.CreatedByAPIKeyID.Int64 != 0 {
		cols = append(cols, "CreatedByAPIKeyID")
		b = append(b, a.CreatedByAPIKeyID.Int64)
	} else {
		err = errors.New("unknown if we are saving activity log for user or api")
		return
	}

	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableActivityLog + `(` + colString + `) VALUES (` + valString + `)`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, b...)
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		return
	}

	a.ID = id
	return
}

// GetActivityLog looks up the latest activites in the activity log. The results can
// be filter by a specific user, a specific API key, an endpoint, and/or a string in
// the form values sent in a request (GET or POST) via a wildcard LIKE. The results
// can be limited by a date range or a limit, with the latest 200 rows being returned
// by default if a date range nor a limit is provided.
func GetActivityLog(ctx context.Context, userID, apiKeyID int64, endpoint, searchFor, startDate, endDate string, numRows uint16) (aa []ActivityLog, err error) {
	const defaultMaxRows uint16 = 200
	if numRows <= 0 {
		numRows = defaultMaxRows
	}

	//Build columns.
	offset := config.GetTimezoneOffsetForSQLite()
	cols := sqldb.Columns{
		TableActivityLog + ".ID",
		TableActivityLog + ".Method",
		TableActivityLog + ".TimeDuration",
		TableActivityLog + ".URL",
		TableActivityLog + ".PostFormValues",
		TableActivityLog + ".Referrer",
		TableActivityLog + ".DatetimeCreated",
		"IFNULL(" + TableUsers + ".Username, '') AS Username",
		"IFNULL(" + TableAPIKeys + ".K, '') AS APIKeyK",
		"IFNULL(" + TableAPIKeys + ".Description, '') AS APIKeyDescription",

		`datetime(` + TableActivityLog + `.DatetimeCreated, '` + offset + `') AS DatetimeCreatedInTZ`,
	}
	colString, err := cols.ForSelect()
	if err != nil {
		return
	}

	//Build query.
	q := `
		SELECT ` + colString + ` 
		FROM ` + TableActivityLog + `
		LEFT JOIN ` + TableUsers + ` ON ` + TableUsers + `.ID=` + TableActivityLog + `.CreatedByUserID
		LEFT JOIN ` + TableAPIKeys + ` ON ` + TableAPIKeys + `.ID=` + TableActivityLog + `.CreatedByAPIKeyID
	`

	var wheres []string
	var b sqldb.Bindvars
	if userID > 0 {
		w := ` (` + TableActivityLog + `.CreatedByUserID = ?)`
		wheres = append(wheres, w)
		b = append(b, userID)
	}
	if apiKeyID > 0 {
		w := ` (` + TableActivityLog + `.CreatedByAPIKeyID = ?)`
		wheres = append(wheres, w)
		b = append(b, userID)
	}
	if endpoint != "" {
		w := ` (` + TableActivityLog + `.URL = ?) `
		wheres = append(wheres, w)
		b = append(b, endpoint)
	}
	if searchFor != "" {
		w := ` (` + TableActivityLog + `.PostFormValues LIKE ?)`
		wheres = append(wheres, w)
		b = append(b, "%"+searchFor+"%")
	}

	useDateRange := false
	if startDate != "" && endDate != "" {
		w := `(DATE(datetime(` + TableActivityLog + `.DatetimeCreated, '` + offset + `')) BETWEEN ? AND ?)`

		wheres = append(wheres, w)
		b = append(b, startDate, endDate)
		useDateRange = true
	}

	if len(wheres) > 0 {
		q += " WHERE " + strings.Join(wheres, " AND ")
	}

	q += ` ORDER BY ` + TableActivityLog + `.TimestampCreated DESC`

	if !useDateRange {
		q += ` LIMIT ` + strconv.FormatInt(int64(numRows), 10)
	}

	//Run query.
	c := sqldb.Connection()
	err = c.SelectContext(ctx, &aa, q, b...)
	return
}

// ClearActivityLog deletes rows from the activity log table prior to a given date
func ClearActivityLog(ctx context.Context, date string) (rowsDeleted int64, err error) {
	q := `
		DELETE FROM ` + TableActivityLog + ` 
		WHERE DatetimeCreated < ?
	`

	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, date)
	if err != nil {
		return
	}

	rowsDeleted, err = res.RowsAffected()
	if err != nil {
		return
	}

	//Not VACUUMing, call VACUUM manually since it may block the db for a while.
	return
}
