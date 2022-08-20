package db

import (
	"context"
	"strings"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/sqldb/v2"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v3"
)

//This table stores notes for each license. We don't just store notes in a note
//column for the license so that we can add more than once note for each license.
//Some notes are added automatically, such as when disabling a license, but users
//can also add notes manually.

// TableLicenseNotes is the name of the table.
const TableLicenseNotes = "license_notes"

// LicenseNote is used to interact with the table.
type LicenseNote struct {
	ID               int64
	DatetimeCreated  string
	DatetimeModified string
	Active           bool

	//a note can be added by a user or via an api call.
	CreatedByUserID   null.Int
	CreatedByAPIKeyID null.Int

	LicenseID int64
	Note      string

	//Calculated fields
	DatetimeCreatedTZ string //DatetimeCreated converted to timezone per config file.
	Timezone          string //extra data for above fields for displaying in GUI.

	//JOINed fields
	CreatedByUsername          null.String
	CreatedByAPIKeyDescription null.String
}

const (
	createTableLicenseNotes = `
		CREATE TABLE IF NOT EXISTS ` + TableLicenseNotes + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified TEXT DEFAULT CURRENT_TIMESTAMP,
			Active INTEGER NOT NULL DEFAULT 1,
			
			CreatedByUserID INTEGER DEFAULT NULL,
			CreatedByAPIKeyID INTEGER DEFAULT NULL,

			LicenseID INTEGER NOT NULL,
			Note TEXT NOT NULL,

			FOREIGN KEY (CreatedByUserID) REFERENCES ` + TableUsers + `(ID),
			FOREIGN KEY (CreatedByAPIKeyID) REFERENCES ` + TableAPIKeys + `(ID),
			FOREIGN KEY (LicenseID) REFERENCES ` + TableLicenses + `(ID)
		)
	`
)

// Validate handles sanitizing and validation before a note is saved.
func (n *LicenseNote) Validate() (errMsg string) {
	//Sanitize.
	n.Note = strings.TrimSpace(n.Note)

	//Validate.
	if n.Note == "" {
		errMsg = "You must provide a note."
		return
	}
	if n.LicenseID == 0 {
		errMsg = "Could not determine what license this note is for."
		return
	}
	if n.CreatedByUserID.Int64 == 0 && n.CreatedByAPIKeyID.Int64 == 0 {
		errMsg = "Could not determine who is creating this note."
		return
	}

	return
}

// Insert saves a note.
// A transaction is optional, pass nil if you don't have one. A transaction is
// used when we are performing an action to a license, for example disabling, and
// we also want to save a note.
func (n *LicenseNote) Insert(ctx context.Context, tx *sqlx.Tx) (err error) {
	//use tx if given, otherwise generate a tx
	txProvided := true
	if tx == nil {
		c := sqldb.Connection()
		tx, err = c.BeginTxx(ctx, nil)
		if err != nil {
			return
		}
		defer tx.Rollback()

		txProvided = false
	}

	//Build columns.
	cols := sqldb.Columns{
		"LicenseID",
		"Note",
	}
	b := sqldb.Bindvars{
		n.LicenseID,
		n.Note,
	}

	if n.CreatedByUserID.Int64 > 0 {
		cols = append(cols, "CreatedByUserID")
		b = append(b, n.CreatedByUserID.Int64)
	} else {
		cols = append(cols, "CreatedByAPIKeyID")
		b = append(b, n.CreatedByAPIKeyID.Int64)
	}

	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableLicenseNotes + `(` + colString + `) VALUES (` + valString + `)`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return
	}

	res, err := stmt.ExecContext(ctx, b...)
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	n.ID = id

	//finish tx if we generated it in this func
	if !txProvided {
		err = tx.Commit()
		if err != nil {
			return
		}
	}

	return
}

// GetNotes looks up the notes for a license.
func GetNotes(ctx context.Context, licenseID int64, orderBy string) (nn []LicenseNote, err error) {
	//base query
	q := `
		SELECT 
			` + TableLicenseNotes + `.*,
			` + TableUsers + `.Username AS CreatedByUsername,
			` + TableAPIKeys + `.Description AS CreatedByAPIKeyDescription 
		FROM ` + TableLicenseNotes + `
		LEFT JOIN ` + TableUsers + ` ON ` + TableUsers + `.ID=` + TableLicenseNotes + `.CreatedByUserID 
		LEFT JOIN ` + TableAPIKeys + ` ON ` + TableAPIKeys + `.ID=` + TableLicenseNotes + `.CreatedByAPIKeyID 
		WHERE 
			(` + TableLicenseNotes + `.LicenseID = ?)
			AND
			(` + TableLicenseNotes + `.Active = ?)
	`
	q += orderBy

	b := sqldb.Bindvars{
		licenseID,
		true, //Active
	}

	c := sqldb.Connection()
	err = c.SelectContext(ctx, &nn, q, b...)

	//Convert datetime to config timezone. This timezone is usually more user
	//friendly than the UTC timezone the datetime is stored as. This conversion
	//isn't handled in SQL since MariaDB and SQLite differ in how they can convert
	//datetimes to different timezones (even though we only support SQLite
	//currently).
	for k, v := range nn {
		nn[k].DatetimeCreatedTZ = GetDatetimeInConfigTimezone(v.DatetimeCreated)
		nn[k].Timezone = config.Data().Timezone
	}
	return
}
