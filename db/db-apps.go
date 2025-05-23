package db

import (
	"context"
	"database/sql"
	"strings"

	"github.com/c9845/licensekeys/v3/licensefile"
	"github.com/c9845/licensekeys/v3/timestamps"
	"github.com/c9845/sqldb/v3"
)

//Apps stores the applications you generate license keys for.
//
//Each app defines a format for the license file used when creating the signature
//and saving the license file. This is saved with each license file's data in case
//changes are made in the future. This ensures that a license file can be
//redownloaded at any time and remain the same as it was previously. Changing these
//fields isn't often used, but is useful in cases where you update your app to handle
//a change for some reason.

// TableApps is the name of the table.
const TableApps = "apps"

// App is used to interact with the table.
type App struct {
	ID               int64
	DatetimeCreated  string
	DatetimeModified string
	CreatedByUserID  int64
	Active           bool

	Name             string
	DaysToExpiration int                    //the number of days to add on to "today" to calculate a default expiration date of a license
	FileFormat       licensefile.FileFormat //yaml, json, etc. the format of the data stored in the text file.
	ShowLicenseID    bool                   //if the ID field of a created license file will be populated/non-zero.
	ShowAppName      bool                   //if the Application field of a created license file will be populated/non-blank.

	//DownloadFilename is the name of the license file when downloaded. This defaults to
	//"appname-license.txt" but can be customized using {{}} placeholders.
	//Placeholders:
	// - {ext} is replaced with the file format's extension prepended by a period (ex: .json);
	// - {appName} is replaced with the app's name, in lowercase and with spaces replaced by underscores.
	// - {licenseID} is replaced with the license's ID.
	DownloadFilename string
}

const (
	createTableApps = `
		CREATE TABLE IF NOT EXISTS ` + TableApps + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified TEXT DEFAULT CURRENT_TIMESTAMP,
			CreatedByUserID INTEGER NOT NULL,
			Active INTEGER NOT NULL DEFAULT 1,

			Name TEXT NOT NULL,
			DaysToExpiration INTEGER NOT NULL DEFAULT 0,
			FileFormat TEXT NOT NULL,
			ShowLicenseID INTEGER NOT NULL DEFAULT 1,
			ShowAppName INTEGER NOT NULL DEFAULT 1,
			DownloadFilename TEXT NOT NULL,

			FOREIGN KEY (CreatedByUserID) REFERENCES ` + TableUsers + `(ID)
		)
	`
)

// Validate is used to validate a struct's data before adding or saving changes. This also
// handles sanitizing.
func (a *App) Validate(ctx context.Context) (errMsg string, err error) {
	//Sanitize
	a.Name = strings.TrimSpace(a.Name)
	a.DownloadFilename = strings.TrimSpace(a.DownloadFilename)
	a.DownloadFilename = strings.ReplaceAll(a.DownloadFilename, " ", "_")

	//Validate
	if a.Name == "" {
		errMsg = "You must provide the name for your app."
		return
	}
	if a.DaysToExpiration < 0 {
		errMsg = "The default license period cannot be less than 0 days."
		return
	}

	err = a.FileFormat.Valid()
	if err != nil {
		errMsg = "Please choose a file format from the provided options."
		return
	}

	if a.DownloadFilename == "" {
		errMsg = "You must provide the name of the license file as it will be downloaded."
		return
	}

	//Check if an app with this name already exists. We don't want duplicate app names.
	//This uses the ID to handle if we are updating an app (ID is > 0) where the same
	//name would be allowed as long as the IDs match (updating "this" app).
	existing, err := GetAppByName(ctx, a.Name)
	if err == sql.ErrNoRows {
		err = nil
	} else if err != nil {
		//some kind of db error occured
		return
	} else if (a.ID > 0 && a.ID != existing.ID) || (a.ID == 0) {
		//no db error occured, but an existing app was returned. We have to determine
		//if "this" app was returned and we are ok since we are updating it and therefore
		//no duplicate name will result.
		errMsg = "An app with this name already exists."
		return
	}

	return
}

// GetAppByName looks up an app by its name.
func GetAppByName(ctx context.Context, name string) (a App, err error) {
	q := `
		SELECT ` + TableApps + `.*
		FROM ` + TableApps + `
		WHERE Name = ?
	`

	c := sqldb.Connection()
	err = c.GetContext(ctx, &a, q, name)
	return
}

// GetAppByID looks up an app by its ID.
func GetAppByID(ctx context.Context, id int64) (a App, err error) {
	q := `
		SELECT ` + TableApps + `.*
		FROM ` + TableApps + `
		WHERE ID = ?
	`

	c := sqldb.Connection()
	err = c.GetContext(ctx, &a, q, id)
	return
}

// Insert saves an app. You should have already called Validate().
func (a *App) Insert(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"CreatedByUserID",
		"Active",
		"Name",
		"DaysToExpiration",
		"FileFormat",
		"ShowLicenseID",
		"ShowAppName",
		"DownloadFilename",
	}
	b := sqldb.Bindvars{
		a.CreatedByUserID,
		a.Active,
		a.Name,
		a.DaysToExpiration,
		a.FileFormat,
		a.ShowLicenseID,
		a.ShowAppName,
		a.DownloadFilename,
	}
	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableApps + `(` + colString + `) VALUES (` + valString + `)`
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
	a.ID = id
	return
}

// GetApps returns the list of apps optionally filtered by active apps only.
func GetApps(ctx context.Context, activeOnly bool) (aa []App, err error) {
	//base query
	q := `
		SELECT ` + TableApps + `.* 
		FROM ` + TableApps

	//filters
	wheres := []string{}
	b := sqldb.Bindvars{}
	if activeOnly {
		w := `(` + TableApps + `.Active = ?)`
		wheres = append(wheres, w)
		b = append(b, activeOnly)
	}

	if len(wheres) > 0 {
		where := " WHERE " + strings.Join(wheres, " AND ")
		q += where
	}

	//complete query
	q += ` ORDER BY ` + TableApps + `.Active DESC, ` + TableApps + `.Name ASC`

	//run query
	c := sqldb.Connection()
	err = c.SelectContext(ctx, &aa, q, b...)
	return
}

// Update saves changes to an app. You should have already called Validate().
func (a *App) Update(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"DatetimeModified",
		"Active",
		"Name",
		"DaysToExpiration",
		"FileFormat",
		"ShowLicenseID",
		"ShowAppName",
		"DownloadFilename",
	}
	colString, err := cols.ForUpdate()
	if err != nil {
		return
	}

	q := `UPDATE ` + TableApps + ` SET ` + colString + ` WHERE ID = ?`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		timestamps.YMDHMS(),
		a.Active,
		a.Name,
		a.DaysToExpiration,
		a.FileFormat,
		a.ShowLicenseID,
		a.ShowAppName,
		a.DownloadFilename,

		a.ID,
	)
	return
}
