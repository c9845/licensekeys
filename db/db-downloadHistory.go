package db

import (
	"context"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/sqldb/v3"
	"gopkg.in/guregu/null.v3"
)

//This table stores a log of each time a license is downloaded. This is useful
//for auditing purposes. This is somewhat of a duplicate of the activity log
//but only tracks downloads of licenses.

// TableDownloadHistory is the name of the table.
const TableDownloadHistory = "download_history"

// DownloadHistory is used to interact with the table.
type DownloadHistory struct {
	ID               int64
	DatetimeCreated  string //no default in CREATE TABLE so that we can set value using golang since we will also set TimestampCreated using golang.
	TimestampCreated int64  //nanoseconds, for sorting, no default in CREATE TABLE since handling nanoseconds with DEFAULT and SQLite is a pain/impossible.
	LicenseID        int64

	//a license can be creatd by a user or via an api call.
	CreatedByUserID   null.Int
	CreatedByAPIKeyID null.Int

	//Calculated fields
	DatetimeCreatedTZ string //DatetimeCreated converted to timezone per config file.
	Timezone          string //extra data for above fields for displaying in GUI.

	//JOINed fields
	CreatedByUsername          null.String
	CreatedByAPIKeyDescription null.String
}

const (
	createTableDownloadHistory = `
		CREATE TABLE IF NOT EXISTS ` + TableDownloadHistory + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT NOT NULL,
			TimestampCreated INTEGER NOT NULL,
			LicenseID INTEGER NOT NULL,

			CreatedByUserID INTEGER DEFAULT NULL,
			CreatedByAPIKeyID INTEGER DEFAULT NULL,

			FOREIGN KEY (CreatedByUserID) REFERENCES ` + TableUsers + `(ID),
			FOREIGN KEY (CreatedByAPIKeyID) REFERENCES ` + TableAPIKeys + `(ID),
			FOREIGN KEY (LicenseID) REFERENCES ` + TableLicenses + `(ID)
		)
	`
)

// Insert saves the record of a license being downloaded.
func (h *DownloadHistory) Insert(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"DatetimeCreated",
		"TimestampCreated",
		"LicenseID",
	}
	b := sqldb.Bindvars{
		h.DatetimeCreated,
		h.TimestampCreated,
		h.LicenseID,
	}

	if h.CreatedByUserID.Int64 > 0 {
		cols = append(cols, "CreatedByUserID")
		b = append(b, h.CreatedByUserID.Int64)
	} else {
		cols = append(cols, "CreatedByAPIKeyID")
		b = append(b, h.CreatedByAPIKeyID.Int64)
	}

	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableDownloadHistory + `(` + colString + `) VALUES (` + valString + `)`
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
	h.ID = id
	return
}

// GetHistory returns the download history for a license.
func GetHistory(ctx context.Context, licenseID int64, orderBy string) (hh []DownloadHistory, err error) {
	q := `
		SELECT 
			` + TableDownloadHistory + `.*,
			` + TableUsers + `.Username AS CreatedByUsername,
			` + TableAPIKeys + `.Description AS CreatedByAPIKeyDescription
		FROM ` + TableDownloadHistory + `
		LEFT JOIN ` + TableUsers + ` ON ` + TableUsers + `.ID = ` + TableDownloadHistory + `.CreatedByUserID 
		LEFT JOIN ` + TableAPIKeys + ` ON ` + TableAPIKeys + `.ID = ` + TableDownloadHistory + `.CreatedByAPIKeyID 
		WHERE 
			(` + TableDownloadHistory + `.LicenseID = ?)
	`
	q += orderBy

	b := sqldb.Bindvars{
		licenseID,
	}

	c := sqldb.Connection()
	err = c.SelectContext(ctx, &hh, q, b...)

	//Convert datetime to config timezone. This timezone is usually more user
	//friendly than the UTC timezone the datetime is stored as. This conversion
	//isn't handled in SQL since MariaDB and SQLite differ in how they can convert
	//datetimes to different timezones (even though we only support SQLite
	//currently).
	for k, v := range hh {
		hh[k].DatetimeCreatedTZ = GetDatetimeInConfigTimezone(v.DatetimeCreated)
		hh[k].Timezone = config.Data().Timezone
	}

	return
}
