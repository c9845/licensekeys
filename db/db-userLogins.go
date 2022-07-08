package db

import (
	"context"
	"strconv"
	"time"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/timestamps"
	"github.com/c9845/sqldb/v2"
)

//This table stores a history of each time a user logs into the app. This is used for
//maintaining logged in sessions as well as forced-log-outs of users by admins.

//TableUserLogins is the name of the table
const TableUserLogins = "user_logins"

//UserLogin is used to interact with the table
type UserLogin struct {
	ID                 int64 //not stored in cookie because it is easily guessed & incremented to find "next" session
	UserID             int64
	DatetimeCreated    string
	DatetimeModified   string
	RemoteIP           string
	UserAgent          string
	TwoFATokenProvided bool //whether or not a 2FA token was provided upon logging in.

	//This is a random, long value that will be stored in a cookie set for the user to
	//identify the login. This is used, over the ID field, since the ID field can easily
	//be guessed and incremented to find the "next" session. We will use this value, from
	//the cookie, to look up the logged in user's data when needed.
	CookieValue string

	//This is used to force a user to log out, force a users session to be
	//marked as invalid, prior to hitting the expiration. This is useful for
	//forcing users to log out for diagnostics, low level fixing of database,
	//security, etc. This is also used, by setting to false, when a user's
	//password is changed so that all currently logged in sessions need to
	//re-provide the password, again for security.
	Active bool

	//when a user's session will expire. This is reset, extended,
	//each time the user visits a new page on the app.
	Expiration int64

	//JOINed fields
	Username string

	//Calculated fields
	DatetimeCreatedTZ string //DatetimeCreated converted to timezone per config file.
	Timezone          string //extra data for above fields for displaying in GUI.
}

const (
	createTableUserLogins = `
		CREATE TABLE IF NOT EXISTS ` + TableUserLogins + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			UserID INTEGER NOT NULL,
			DatetimeCreated DATETIME DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified DATETIME DEFAULT CURRENT_TIMESTAMP,
			RemoteIP TEXT NOT NULL,
			UserAgent TEXT NOT NULL,
			TwoFATokenProvided INTEGER NOT NULL DEFAULT 0,
			CookieValue TEXT NOT NULL,
			Active INTEGER NOT NULL DEFAULT 1,
			Expiration INTEGER NOT NULL DEFAULT 0,

			FOREIGN KEY(UserID) REFERENCES ` + TableUsers + `(ID)
		)
	`

	createIndexUserLoginsValueUnique     = `CREATE UNIQUE INDEX IF NOT EXISTS ` + TableUserLogins + `__CookieValue_idx ON ` + TableUserLogins + ` (CookieValue)`
	createIndexUserLoginsDatetimeCreated = `CREATE INDEX IF NOT EXISTS ` + TableUserLogins + `__DatetimeCreated_idx ON ` + TableUserLogins + ` (DatetimeCreated)`
)

//Insert saves an entry to the database for a user logging in to the app.
//you should have already performed validation.
func (u *UserLogin) Insert(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"UserID",
		"RemoteIP",
		"UserAgent",
		"TwoFATokenProvided",
		"CookieValue",
		"Active",
		"Expiration",
	}
	b := sqldb.Bindvars{
		u.UserID,
		u.RemoteIP,
		u.UserAgent,
		u.TwoFATokenProvided,
		u.CookieValue,
		u.Active,
		u.Expiration,
	}
	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableUserLogins + `(` + colString + `) VALUES (` + valString + `)`

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
	u.ID = id
	return
}

//GetLoginByCookieValue looks up login/session data by the cookie stored value.
func GetLoginByCookieValue(ctx context.Context, cv string) (l UserLogin, err error) {
	c := sqldb.Connection()
	q := `
		SELECT 
			` + TableUserLogins + `.*,
			` + TableUsers + `.Username
		FROM ` + TableUserLogins + `
		JOIN ` + TableUsers + ` ON ` + TableUsers + `.ID=` + TableUserLogins + `.UserID
		WHERE ` + TableUserLogins + `.CookieValue = ?
	`
	err = c.GetContext(ctx, &l, q, cv)
	return
}

//DisableLoginsForUser disables all sessions for a user that are active and not currently
//expired. This is used upon logging in when single sessions is enabled to mark all other
//currently active and non-expired sessions as inactive to enforce the single session policy.
//UserID is required to prevent us from mistakenly disabled all sessions for all users.
func DisableLoginsForUser(ctx context.Context, userID int64) (err error) {
	c := sqldb.Connection()
	q := `
		UPDATE ` + TableUserLogins + `
		SET 
			Active = ?,
			DatetimeModified = ?
		WHERE
			(UserID = ?)
			AND
			(Active = ?)
			AND
			(Expiration > ?)
	`
	b := sqldb.Bindvars{
		false,               //setting Active to false
		timestamps.YMDHMS(), //dateime modified

		userID,
		true,              //where Active is currently true, just a little query optimization to ignore already inactive sessions/logins.
		time.Now().Unix(), //to find expiration timestamps in the future because we can ignore ones in the past
	}

	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	_, err = stmt.ExecContext(ctx, b...)
	return
}

//ExtendLoginExpiration updates the expiration timestamp for a user's login. This is used to
//reset the time a session will expire to keep users logged in if they are active within
//the app.
func (l *UserLogin) ExtendLoginExpiration(ctx context.Context, newExpiration int64) (err error) {
	c := sqldb.Connection()
	q := `
		UPDATE ` + TableUserLogins + ` 
		SET 
			Expiration = ?,
			DatetimeModified = ?
		WHERE
			CookieValue = ?
	`
	b := sqldb.Bindvars{
		newExpiration,
		timestamps.YMDHMS(),

		l.CookieValue,
	}

	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	_, err = stmt.ExecContext(ctx, b...)
	return
}

//GetUserLogins looks up successful logins
//This defaults to looking up the last 200 rows.  Set to 0 for 200 rows.
//You can also filter by user.  Set to 0 for all users.
func GetUserLogins(ctx context.Context, userID int64, numRows uint16) (uu []UserLogin, err error) {
	const defaultMaxRows uint16 = 200

	//build columns
	cols := sqldb.Columns{
		TableUserLogins + `.ID`,
		TableUserLogins + `.UserID`,
		TableUserLogins + `.RemoteIP`,
		TableUserLogins + `.UserAgent`,
		`IFNULL(` + TableUserLogins + `.TwoFATokenProvided, false) AS TwoFATokenProvided`,

		`IFNULL(` + TableUsers + `.Username, "") AS Username`, //why is this IFNULL???
	}

	//ensure datetimes are returned in same format, yyyy-mm-dd hh:mm:ss, regardless of db type.
	//This handles the golang sqlite driver returning datetimes in yyyy-mm-ddThh:mm:ssZ.
	//We are better off returning the same value from the db/driver rather then having to handle
	//multiple different formats in GetDatetimeInConfigTimezone() or elsewhere.
	if sqldb.IsMariaDB() {
		cols = append(cols, TableUserLogins+".DatetimeCreated")
	} else if sqldb.IsSQLite() {
		cols = append(cols, "datetime("+TableUserLogins+".DatetimeCreated) AS DatetimeCreated")
	}

	colString, err := cols.ForSelect()
	if err != nil {
		return
	}

	//build query
	q := `
		SELECT ` + colString + ` 
		FROM ` + TableUserLogins + `
		LEFT JOIN ` + TableUsers + ` ON ` + TableUsers + `.ID=` + TableUserLogins + `.UserID
	`

	b := sqldb.Bindvars{}
	if userID > 0 {
		q += ` WHERE (` + TableUserLogins + `.UserID = ?)`
		b = append(b, userID)
	}

	q += ` ORDER BY ` + TableUserLogins + `.DatetimeCreated DESC`
	q += ` LIMIT `
	if numRows > 0 {
		q += strconv.FormatInt(int64(numRows), 10)
	} else {
		q += strconv.FormatInt(int64(defaultMaxRows), 10)
	}

	//run query
	c := sqldb.Connection()
	err = c.SelectContext(ctx, &uu, q, b...)

	//handle converting datetimes to correct timezone
	//This isn't handled in sql query since mariadb and sqlite differ in how they can
	//convert a datetime to a different timezone.  Doing it in this manner ensures the
	//same conversion method is applied so golang does the conversion.
	for k, v := range uu {
		uu[k].DatetimeCreatedTZ = GetDatetimeInConfigTimezone(v.DatetimeCreated)
		uu[k].Timezone = config.Data().Timezone
	}

	return
}
