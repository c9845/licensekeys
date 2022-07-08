package db

//This table stores a history of user and API interactions with this app. This is meant
//for use for auditing purposes.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/timestamps"
	"github.com/c9845/sqldb/v2"
	"gopkg.in/guregu/null.v3"
)

//TableActivityLog is the name of the table
const TableActivityLog = "activity_log"

//ActivityLog is used to interact with the table
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

	//either CreatedByUserID or CreatedByAPIKeyID is provided, never both
	CreatedByUserID   null.Int
	CreatedByAPIKeyID null.Int

	//JOINed fields
	Username          string
	APIKeyDescription string
	APIKeyK           string //the actual api key

	//Calculated fields
	DatetimeCreatedTZ string //DatetimeCreated converted to timezone per config file.
	Timezone          string //extra data for above fields for displaying in GUI.
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
			
			CreatedByUserID INTEGER DEFAULT NULL,
			CreatedByAPIKeyID INTEGER DEFAULT NULL,
			
			FOREIGN KEY(CreatedByUserID) REFERENCES ` + TableUsers + `(ID),
			FOREIGN KEY(CreatedByAPIKeyID) REFERENCES ` + TableAPIKeys + `(ID)
		)
	`

	createIndexActivityLogTimestampCreated = `CREATE INDEX IF NOT EXISTS ` + TableActivityLog + `__TimestampCreated_idx ON ` + TableActivityLog + ` (TimestampCreated)`
	createIndexActivityLogDatetimeCreated  = `CREATE INDEX IF NOT EXISTS ` + TableActivityLog + `__DatetimeCreated_idx ON ` + TableActivityLog + ` (DatetimeCreated)`
)

//Insert saves a log entry to the database for an action performed
//by a user or via an api key.
//you should have already performed validation.
func (a *ActivityLog) Insert(ctx context.Context) (err error) {
	//similar fields between user actions and api actions
	cols := sqldb.Columns{
		"Method",
		"URL",
		"RemoteIP",
		"UserAgent",
		"TimeDuration",
		"PostFormValues",
		"DatetimeCreated",
		"TimestampCreated",
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
	}

	//add fields based on if this action was caused by user or api
	if a.CreatedByUserID.Int64 > 0 {
		cols = append(cols, "CreatedByUserID")
		b = append(b, a.CreatedByUserID.Int64)
	} else if a.CreatedByAPIKeyID.Int64 != 0 {
		cols = append(cols, "CreatedByAPIKeyID")
		b = append(b, a.CreatedByAPIKeyID.Int64)
	} else {
		err = fmt.Errorf("unknown if we are saving activity log for user or api. %w", ErrInputInvalid)
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

	res, err := stmt.ExecContext(ctx, b...)
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	a.ID = id
	return
}

//GetActivityLog looks up the activities in the log.  This defaults
//to returning the last 200 runs if numRows is 0.  This can filter
//results by user and/or by a search string which performs a wildcard-ed
//LIKE query on the form values.
func GetActivityLog(ctx context.Context, userID int64, endpoint, searchFor string, numRows uint16) (aa []ActivityLog, err error) {
	const defaultMaxRows uint16 = 200

	//build columns
	cols := sqldb.Columns{
		TableActivityLog + ".ID",
		TableActivityLog + ".Method",
		TableActivityLog + ".TimeDuration",
		TableActivityLog + ".URL",
		TableActivityLog + ".PostFormValues",
		"IFNULL(" + TableUsers + ".Username, '') AS Username",
		"IFNULL(" + TableAPIKeys + ".K, '') AS APIKeyK",
		"IFNULL(" + TableAPIKeys + ".Description, '') AS APIKeyDescription",
	}

	//ensure datetimes are returned in same format, yyyy-mm-dd hh:mm:ss, regardless of db type.
	//This handles the golang sqlite driver returning datetimes in yyyy-mm-ddThh:mm:ssZ.
	//We are better off returning the same value from the db/driver rather then having to handle
	//multiple different formats in GetDatetimeInConfigTimezone() or elsewhere.
	if sqldb.IsMariaDB() {
		cols = append(cols, TableActivityLog+".DatetimeCreated")
	} else if sqldb.IsSQLite() {
		cols = append(cols, "datetime("+TableActivityLog+".DatetimeCreated) AS DatetimeCreated")
	}

	colString, err := cols.ForSelect()
	if err != nil {
		return
	}

	//build query
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

	if len(wheres) > 0 {
		q += " WHERE " + strings.Join(wheres, " AND ")
	}

	q += ` ORDER BY ` + TableActivityLog + `.TimestampCreated DESC`
	q += ` LIMIT `
	if numRows > 0 {
		q += strconv.FormatInt(int64(numRows), 10)
	} else {
		q += strconv.FormatInt(int64(defaultMaxRows), 10)
	}

	//run query
	c := sqldb.Connection()
	err = c.SelectContext(ctx, &aa, q, b...)

	// log.Println(q, b, searchFor)

	//handle converting datetimes to correct timezone
	//This isn't handled in sql query since mariadb and sqlite differ in how they can
	//convert a datetime to a different timezone.  Doing it in this manner ensures the
	//same conversion method is applied so golang does the conversion.
	for k, v := range aa {
		aa[k].DatetimeCreatedTZ = GetDatetimeInConfigTimezone(v.DatetimeCreated)
		aa[k].Timezone = config.Data().Timezone
	}

	return
}

//ClearActivityLog deletes rows from the activity log table prior to a given date
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

	res, err := stmt.ExecContext(ctx, date)
	if err != nil {
		return
	}

	rowsDeleted, err = res.RowsAffected()
	return
}
