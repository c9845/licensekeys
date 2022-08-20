package db

//App settings are settings that modify functionality of the app or change how
//the GUI is displayed. There should only ever be one row in this table.

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/c9845/licensekeys/v2/timestamps"
	"github.com/c9845/sqldb/v2"
	"github.com/jmoiron/sqlx"
)

// TableAppSettings is the name of the table
const TableAppSettings = "app_settings"

// AppSettings is used to interact with the table
type AppSettings struct {
	ID               int64
	DatetimeModified string

	EnableActivityLogging bool //whether or not the app tracks user activity (page views, endpoints, etc.)
	AllowAPIAccess        bool //whether or not external access to this app is allowed
	Allow2FactorAuth      bool //if 2 factor authentication can be used
	Force2FactorAuth      bool //if all users are required to have 2 factor auth enabled prior to logging in (check if at least one user has 2fa enabled first to prevent lock out!)
	ForceSingleSession    bool //user can only be logged into the app in one browser at a time. used as a security tool.
}

const (
	createTableAppSettings = `
		CREATE TABLE IF NOT EXISTS ` + TableAppSettings + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeModified TEXT DEFAULT CURRENT_TIMESTAMP,
			
			EnableActivityLogging INTEGER NOT NULL DEFAULT 1,
			AllowAPIAccess INTEGER NOT NULL DEFAULT 1,
			Allow2FactorAuth INTEGER NOT NULL DEFAULT 0,
			Force2FactorAuth INTEGER NOT NULL DEFAULT 0,
			ForceSingleSession INTEGER NOT NULL DEFAULT 1
		)
	`
)

func insertInitialAppSettings(c *sqlx.DB) (err error) {
	//check if initial data already exists
	ctx := context.Background()
	_, err = GetAppSettings(ctx)
	if err == nil {
		log.Println("insertInitialAppSettings...already exists")
		return
	} else if err != nil && err != sql.ErrNoRows {
		return
	}

	cols := sqldb.Columns{
		"EnableActivityLogging",
		"AllowAPIAccess",
		"Allow2FactorAuth",
		"Force2FactorAuth",
		"ForceSingleSession",
	}
	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	//insert initial data
	q := `INSERT INTO ` + TableAppSettings + `(` + colString + `) VALUES (` + valString + `)`
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}

	_, err = stmt.ExecContext(
		ctx,

		true,  //EnableActivityLogging
		false, //AllowAPIAccess
		true,  //Allow2FactorAuth
		false, //Force2FactorAuth
		false, //ForceSingleSession
	)
	return
}

// errors specific to app settings
var (
	//ErrAppSettingDisabled is the generic error used when an app setting is turned of
	//A more detailed or specific err can be defined for certain use cases.
	ErrAppSettingDisabled = errors.New("app setting is turned off")

	//ErrNoUser2FAEnabled is returned when a user tries to force turning on 2fa but no admin user has 2fa set up yet
	//You can't force the requirement of 2fa if no user has 2fa set up yet otherwise you will be locked out of the app.
	ErrNoUser2FAEnabled = errors.New("no user has 2 factor auth enabled")
)

// GetAppSettings looks up the current app settings used for this app.
func GetAppSettings(ctx context.Context) (a AppSettings, err error) {
	//get data from database
	q := `
		SELECT *
		FROM ` + TableAppSettings + `
		LIMIT 1
	`
	c := sqldb.Connection()
	err = c.GetContext(ctx, &a, q)

	return
}

// Update updates the saved app settings to the given values
func (a *AppSettings) Update(ctx context.Context) (err error) {

	//if forcing 2fa is turned on, make sure 2fa is enabled too
	//but have to make sure at least one user has 2fa enabled first!
	if a.Force2FactorAuth {
		q := `
			SELECT COUNT(ID)
			FROM ` + TableUsers + `
			WHERE
				Active = ?
				AND
				TwoFactorAuthEnabled = ?
				AND
				Administrator = ?
		`
		var count int
		c := sqldb.Connection()
		err = c.GetContext(ctx, &count, q, true, true, true)
		if err != nil {
			return
		}
		if count < 1 {
			return ErrNoUser2FAEnabled
		}

		a.Allow2FactorAuth = true
	}

	//update app settings
	cols := sqldb.Columns{
		"DatetimeModified",

		"EnableActivityLogging",
		"AllowAPIAccess",
		"Allow2FactorAuth",
		"Force2FactorAuth",
		"ForceSingleSession",
	}
	colString, err := cols.ForUpdate()
	if err != nil {
		return
	}

	q := `UPDATE ` + TableAppSettings + ` SET ` + colString

	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}

	_, err = stmt.ExecContext(
		ctx,

		timestamps.YMDHMS(),

		a.EnableActivityLogging,
		a.AllowAPIAccess,
		a.Allow2FactorAuth,
		a.Force2FactorAuth,
		a.ForceSingleSession,
	)

	return
}
