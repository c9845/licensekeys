package db

import (
	"context"
	"strings"

	"github.com/c9845/sqldb/v3"
)

//This table stores browser data for browsers a user has logged into this app on with
//2 Factor Authentication. This is used to remove the need for users to provide a 2FA
//token with each log in to the app.

// TableAuthorizedBrowsers is the name of the table
const TableAuthorizedBrowsers = "user_authorized_browsers"

// AuthorizedBrowser is used to interact with the table
type AuthorizedBrowser struct {
	ID              int64
	DatetimeCreated string
	UserID          int64
	Active          bool

	RemoteIP  string
	UserAgent string
	Cookie    string //a token saved to user's cookie so that if user resets browser we reprompt for 2fa token
	Timestamp int64  //so we can force asking for a 2fa code after so much time regardless if browser is trusted
}

const (
	createTableAuthorizedBrowsers = `
		CREATE TABLE IF NOT EXISTS ` + TableAuthorizedBrowsers + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			UserID INTEGER NOT NULL,
			Active INTEGER NOT NULL DEFAULT 1,

			RemoteIP TEXT NOT NULL,
			UserAgent TEXT NOT NULL,
			Cookie TEXT NOT NULL,
			Timestamp INTEGER NOT NULL,

			FOREIGN KEY(UserID) REFERENCES ` + TableUsers + ` (ID)
		)
	`

	createIndexAuthorizedBrowsersRemoteIP     = `CREATE INDEX IF NOT EXISTS ` + TableAuthorizedBrowsers + `__RemoteIP_idx ON ` + TableAuthorizedBrowsers + ` (RemoteIP)`
	createIndexAuthorizedBrowsersCookieUnique = `CREATE UNIQUE INDEX IF NOT EXISTS ` + TableAuthorizedBrowsers + `__Cookie_idx ON ` + TableAuthorizedBrowsers + ` (Cookie)`
	createIndexAuthorizedBrowsersCookie       = `CREATE INDEX IF NOT EXISTS ` + TableAuthorizedBrowsers + `__Cookie_idx2 ON ` + TableAuthorizedBrowsers + ` (Cookie)`
)

// Insert saves a row to the database.
func (a *AuthorizedBrowser) Insert(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"UserID",
		"RemoteIP",
		"UserAgent",
		"Cookie",
		"Timestamp",
	}
	b := sqldb.Bindvars{
		a.UserID,
		a.RemoteIP,
		a.UserAgent,
		a.Cookie,
		a.Timestamp,
	}
	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableAuthorizedBrowsers + `(` + colString + `) VALUES (` + valString + `)`

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

// GetAuthorizedBrowser looks up an authorized browser for a user by their ID, the
// client's IP address, and the 2FA cookie value. This is used to check if the user
// has already provided a 2FA token recently for the system they are trying to log in
// from.
//
// IP address and cookie are used to identify browser. We used to use useragent as
// part of identifying browser too, but it changes too frequently (browser updates)
// and caused 2FA token to be required a lot.
func GetAuthorizedBrowser(ctx context.Context, userID int64, ip, cookie string, activeOnly bool) (a AuthorizedBrowser, err error) {
	q := `
		SELECT *
		FROM ` + TableAuthorizedBrowsers + `
	`

	var wheres []string
	var b sqldb.Bindvars

	w := `(UserID = ?)`
	wheres = append(wheres, w)
	b = append(b, userID)

	w = `(RemoteIP = ?)`
	wheres = append(wheres, w)
	b = append(b, ip)

	w = `(Cookie = ?)`
	wheres = append(wheres, w)
	b = append(b, cookie)

	if activeOnly {
		w := `(Active = ?)`
		wheres = append(wheres, w)
		b = append(b, true)
	}

	if len(wheres) > 0 {
		q += ` WHERE ` + strings.Join(wheres, " AND ")
	}

	c := sqldb.Connection()
	err = c.GetContext(ctx, &a, q, b...)
	return
}

// DisableAllAuthorizedBrowsers disabled all saved authorized browsers for a user.
// Authorized browsers are saved when 2FA is enabled.  This func is used to "unremember"
// all authorized browsers so user has to provide 2FA token again.  This func is called
// in db-users.go ForceUserLogout().
func DisableAllAuthorizedBrowsers(ctx context.Context, userID int64) (err error) {
	q := `
		UPDATE ` + TableAuthorizedBrowsers + ` 
		SET 
			Active = ? 
		WHERE 
			(UserID = ?)
	`

	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		false, //set active to...

		userID,
	)
	return
}

// ClearAuthorizedBrowsers deletes rows from the authorized browsers table prior to a
// given date.
//
// This is used from the admin tools page to clean up old entries and shrink the
// database size.
func ClearAuthorizedBrowsers(ctx context.Context, date string) (rowsDeleted int64, err error) {
	//Delete rows from table.
	q := `
		DELETE FROM ` + TableAuthorizedBrowsers + ` 
		WHERE 
			(DatetimeCreated < ?)
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
