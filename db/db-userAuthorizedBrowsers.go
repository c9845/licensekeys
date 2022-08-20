package db

import (
	"context"

	"github.com/c9845/sqldb/v2"
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
	RemoteIP        string
	UserAgent       string
	Cookie          string //a token saved to user's cookie so that if user resets browser we reprompt for 2fa token
	Timestamp       int64  //so we can force asking for a 2fa code after so much time regardless if browser is trusted
	Active          bool
}

const (
	createTableAuthorizedBrowsers = `
		CREATE TABLE IF NOT EXISTS ` + TableAuthorizedBrowsers + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			UserID INTEGER NOT NULL,
			RemoteIP TEXT NOT NULL,
			UserAgent TEXT NOT NULL,
			Cookie TEXT NOT NULL,
			Timestamp INTEGER NOT NULL,
			Active INTEGER NOT NULL DEFAULT 1,

			FOREIGN KEY(UserID) REFERENCES ` + TableUsers + ` (ID)
		)
	`

	createIndexAuthorizedBrowsersRemoteIP     = `CREATE INDEX IF NOT EXISTS ` + TableAuthorizedBrowsers + `__RemoteIP_idx ON ` + TableAuthorizedBrowsers + ` (RemoteIP)`
	createIndexAuthorizedBrowsersCookieUnique = `CREATE UNIQUE INDEX IF NOT EXISTS ` + TableAuthorizedBrowsers + `__Cookie_idx ON ` + TableAuthorizedBrowsers + ` (Cookie)`
)

// Insert saves a row to the database.
// you should have already performed validation.
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

	res, err := stmt.ExecContext(ctx, b...)
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	a.ID = id
	return
}

// LookUpAuthorizedBrowser looks up if a browser identified by ip, useragent, and cookie
// has already been authorized via 2fa.
func LookUpAuthorizedBrowser(ctx context.Context, userID int64, ip, ua, cookie string, activeOnly bool) (a AuthorizedBrowser, err error) {
	//Note, useragent was removed from checking because it changes too frequently and
	//was causing users to provide their 2fa token often. This is mostly due to browsers
	//updating frequently and including the browser version number in the useragent string.
	q := `
		SELECT *
		FROM ` + TableAuthorizedBrowsers + `
		WHERE
			UserID = ?
			AND
			RemoteIP = ?
			AND
			Cookie = ?
	`
	b := sqldb.Bindvars{userID, ip, ua, cookie}

	if activeOnly {
		q += ` AND Active = ?`
		b = append(b, true)
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
	//filtering by Active = true so we don't look up rows that
	//are already set to false.
	q := `
		UPDATE ` + TableAuthorizedBrowsers + ` 
		SET Active = ? 
		WHERE 
			Active = ?
			AND
			UserID = ?
	`

	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}

	_, err = stmt.ExecContext(
		ctx,

		false, //set active to...
		true,  //where active is...

		userID,
	)
	return
}
