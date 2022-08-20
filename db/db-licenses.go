package db

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/licensefile"
	"github.com/c9845/sqldb/v2"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v3"
)

//This table keeps a record of each license that was generated and the common data
//for each license. A license is created once, and the signature is generated once,
//so we need to keep a copy of each piece of data included in a license so that a
//license can be downloaded at any time in the future. The signature is NOT created
//each time a license is downloaded. This is done so that each time a license is
//downloaded, it is the exact same. Doing this stops us from needing to recreate the
//signature each time and keeping a record of each.
//
//You would use the data in this table in conjunction with the custom_field_results
//table to "rebuild" a license to allow for redownloading it.

// TableLicenses is the name of the table.
const TableLicenses = "licenses"

// License is used to interact with the table.
type License struct {
	ID               int64
	DatetimeCreated  string
	DatetimeModified string
	Active           bool

	//a license can be created by a user or via an api call.
	CreatedByUserID   null.Int
	CreatedByAPIKeyID null.Int

	//details chosen/provided in gui by user
	KeyPairID      int64  //the keypair used to sign the license
	CompanyName    string //company this license is for
	ContactName    string //who requested this license
	PhoneNumber    string //contact info
	Email          string //contact info
	IssueDate      string //yyyy-mm-dd format
	IssueTimestamp int64  //unix timestamp in seconds
	ExpireDate     string //yyyy-mm-dd format

	//The signature generated using the private key from the keypair. This is
	//generated once when the license is first created using the the common
	//license details and the common field results stored in the app's file
	//format. This is set to "" when license data is saved so we can get a
	//license ID before the license data is signed since some licenses contain
	//the license ID.
	Signature string

	//This is set to true ONLY after a license's data is saved, the signature
	//is created, and we reread the signed license file and check the signature
	//with the public key. This is used to ensure that a license can actually
	//be verified as authentic with the respective public key.
	Verified bool

	//These fields are copied from the app's details when the license
	//is created.
	AppName       string
	FileFormat    licensefile.FileFormat
	ShowLicenseID bool
	ShowAppName   bool

	//Calculated fields
	Expired           bool   //true if Expire date is greater than current date
	DatetimeCreatedTZ string //DatetimeCreated converted to timezone per config file.
	IssueDateTZ       string // " " " "
	Timezone          string //extra data for above fields for displaying in GUI.

	//JOINed fields
	KeyPairAlgoType            licensefile.KeyPairAlgoType
	CreatedByUsername          null.String
	CreatedByAPIKeyDescription null.String
	AppID                      int64
	AppFileFormat              licensefile.FileFormat
	AppDownloadFilename        string
	RenewedFromLicenseID       null.Int
	RenewedToLicenseID         null.Int
}

const (
	createTableLicenses = `
		CREATE TABLE IF NOT EXISTS ` + TableLicenses + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified TEXT DEFAULT CURRENT_TIMESTAMP,
			Active INTEGER NOT NULL DEFAULT 1,
			
			CreatedByUserID INTEGER DEFAULT NULL,
			CreatedByAPIKeyID INTEGER DEFAULT NULL,

			KeyPairID INTEGER NOT NULL,
			CompanyName TEXT NOT NULL,
			ContactName TEXT NOT NULL,
			PhoneNumber TEXT NOT NULL,
			Email TEXT NOT NULL,
			IssueDate TEXT NOT NULL,
			IssueTimestamp INT NOT NULL,
			ExpireDate TEXT NOT NULL,

			Signature TEXT NOT NULL,
			Verified INTEGER NOT NULL DEFAULT 0,

			AppName TEXT NOT NULL,
			FileFormat TEXT NOT NULL,
			ShowLicenseID INTEGER NOT NULL DEFAULT 0,
			ShowAppName INTEGER NOT NULL DEFAULT 0,

			FOREIGN KEY (CreatedByUserID) REFERENCES ` + TableUsers + `(ID),
			FOREIGN KEY (CreatedByAPIKeyID) REFERENCES ` + TableAPIKeys + `(ID),
			FOREIGN KEY (KeyPairID) REFERENCES ` + TableKeyPairs + `(ID)
		)
	`
)

// setLicenseIDStartingValue sets the starting value that the ID will auto increment from
func setLicenseIDStartingValue(c *sqlx.DB) error {
	const startingValue = 10000

	//make sure this hasn't already been run and ID isn't already larger
	q := `
		SELECT 
			IFNULL(MAX(ID), 0) AS ID
		FROM ` + TableLicenses
	var currentMaxID int64
	ctx := context.Background()
	err := c.GetContext(ctx, &currentMaxID, q)
	if err != nil {
		return err
	}

	//Handle if current max ID is already bigger then the value we want to set.
	//This happens when user runs executable with --deploy-db flag even though db
	//is already deployed (i.e. schema updates with new tables being deployed).
	if currentMaxID > startingValue {
		log.Println("setLicenseIDStartingValue...max ID already set to", currentMaxID)
		return nil
	}

	//Update the db to set the new ID to start incrementing from.
	q = `SELECT seq FROM SQLITE_SEQUENCE WHERE name = ?`
	var seq int64
	innerErr := c.GetContext(ctx, &seq, q, createTableLicenses)
	if innerErr == sql.ErrNoRows {
		q = `INSERT INTO SQLITE_SEQUENCE (name, seq) VALUES (?, ?)`
		stmt, innerErr := c.PrepareContext(ctx, q)
		if innerErr != nil {
			return innerErr
		}

		_, innerErr = stmt.ExecContext(ctx, TableLicenses, startingValue)
		return innerErr
	} else if innerErr != nil {
		return innerErr
	}

	return nil
}

// Validate handle sanitizing and validation of the license data. This only
// handle the common fields, not custom fields.
//
// viaAPI alters the validation based on if this function is being called due
// to a license being created by an API request. We have to handle validation
// slightly different due to how the data will be provided in this request.
func (l *License) Validate(ctx context.Context) (errMsg string, err error) {
	//Sanitize.
	l.CompanyName = strings.TrimSpace(l.CompanyName)
	l.ContactName = strings.TrimSpace(l.ContactName)
	l.PhoneNumber = strings.TrimSpace(l.PhoneNumber)
	l.Email = strings.TrimSpace(l.Email)
	l.ExpireDate = strings.TrimSpace(l.ExpireDate)

	//Determine the parent app or keypair used to create this license with.
	//Either the key pair ID or app ID must be provided. If the app ID is provided,
	//then the default key pair will be used.
	//
	//If both an AppID and KeyPairID are provided, we default to using the AppID and
	//look up the default key pair for the app.
	if l.KeyPairID < 1 && l.AppID < 1 {
		errMsg = "Could not determine which app or key pair you want to use for the license."
		return
	} else if l.KeyPairID == 0 && l.AppID > 0 {
		//Request provided an app ID. The license is probably being created via
		//the api, not the GUI. Look up the default keypair for this app.
		defaultKeypair, innerErr := GetDefaultKeyPair(ctx, l.AppID)
		if innerErr != nil {
			err = innerErr
			return
		}
		l.KeyPairID = defaultKeypair.ID
	} else if l.KeyPairID > 0 && l.AppID > 0 {
		//This state can occur during an API request to create a license when the
		//request specifies a key pair ID. When this occurs, we need to look up the
		//related app ID for use in translating the custom field map to a slice of
		//structs.
		_ = "" //just to suppress staticcheck linter warnings
	}

	//Validate.
	if l.CompanyName == "" {
		errMsg = "You must provide the company name for which this license is for."
		return
	}
	if l.ContactName == "" {
		errMsg = "You must provide the contact name of who requested this license."
		return
	}
	if l.PhoneNumber == "" {
		errMsg = "You must provide a phone number."
		return
	}
	if l.Email == "" {
		//We don't check if an email is a valid email here, like we do client side
		//since we cannot guarantee that the regexp will be parsed exactly the same
		//and we don't want the server to catch an "invalid" email but the client
		//side not, creating a mismatch in errors and possible confusion for the
		//user submitting the data.
		errMsg = "You must provide an email address."
		return
	}
	if l.ExpireDate == "" {
		errMsg = "You must provide an expiration date for the license."
		return
	}

	//Make sure expiration date is in the future.
	expDate, err := time.Parse("2006-01-02", l.ExpireDate)
	if err != nil {
		return
	}
	if !expDate.After(time.Now()) {
		errMsg = "The expiration date must be in the future."
		return
	}

	return
}

// Insert saves a license. You should have already called Validate().
// You need to validate the license and update the Signature and
// Verified fields.
func (l *License) Insert(ctx context.Context, tx *sqlx.Tx) (err error) {
	if l.CreatedByUserID.Int64 == 0 && l.CreatedByAPIKeyID.Int64 == 0 {
		return errors.New("cannot determine how license is being added")
	}

	cols := sqldb.Columns{
		"DatetimeCreated",
		"Active",

		"KeyPairID",
		"CompanyName",
		"ContactName",
		"PhoneNumber",
		"Email",
		"IssueDate",
		"IssueTimestamp",
		"ExpireDate",

		"Signature", //always "" when license is first saved until data is verified
		"Verified",  //always false when license is first saved until data is read back from db and checked

		"AppName",
		"FileFormat",
		"ShowLicenseID",
		"ShowAppName",
	}
	b := sqldb.Bindvars{
		l.DatetimeCreated,
		true, //Active

		l.KeyPairID,
		l.CompanyName,
		l.ContactName,
		l.PhoneNumber,
		l.Email,
		l.IssueDate,
		l.IssueTimestamp,
		l.ExpireDate,

		"",    //Signature
		false, //Verified

		l.AppName,
		l.FileFormat,
		l.ShowLicenseID,
		l.ShowAppName,
	}

	if l.CreatedByUserID.Int64 > 0 {
		cols = append(cols, "CreatedByUserID")
		b = append(b, l.CreatedByUserID.Int64)
	} else {
		cols = append(cols, "CreatedByAPIKeyID")
		b = append(b, l.CreatedByAPIKeyID.Int64)
	}

	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableLicenses + `(` + colString + `) VALUES (` + valString + `)`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return
	}

	res, err := stmt.ExecContext(ctx, b...)
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	l.ID = id
	return
}

// SaveSignature updates a saved license by saving the generated signature.
func (l *License) SaveSignature(ctx context.Context, tx *sqlx.Tx) (err error) {
	q := `
		UPDATE ` + TableLicenses + ` 
		SET Signature = ?
		WHERE ID = ?
	`

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return
	}

	_, err = stmt.ExecContext(ctx, l.Signature, l.ID)
	return
}

// MarkVerified updates a saved license by marking it as valid.
func (l *License) MarkVerified(ctx context.Context) (err error) {
	q := `
		UPDATE ` + TableLicenses + ` 
		SET Verified = ?
		WHERE ID = ?
	`

	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}

	_, err = stmt.ExecContext(ctx, l.Verified, l.ID)
	return
}

// GetLicenses looks up a list of licenses optionally filtered by app and active
// licenses only.
func GetLicenses(ctx context.Context, appID, limit int64, activeOnly bool, columns sqldb.Columns) (ll []License, err error) {
	//build base query
	cols, err := columns.ForSelect()
	if err != nil {
		return
	}

	q := `
		SELECT ` + cols + ` 
		FROM ` + TableLicenses + ` 
		JOIN ` + TableKeyPairs + ` ON ` + TableKeyPairs + `.ID=` + TableLicenses + `.KeyPairID 
		JOIN ` + TableApps + ` ON ` + TableApps + `.ID=` + TableKeyPairs + `.AppID

		LEFT JOIN ` + TableRenewalRelationships + ` AS rrFrom ON rrFrom.FromLicenseID = ` + TableLicenses + `.ID
		LEFT JOIN ` + TableRenewalRelationships + ` AS rrTo   ON rrTo.ToLicenseID = ` + TableLicenses + `.ID
	`

	//filters
	wheres := []string{}
	b := sqldb.Bindvars{}
	if appID > 0 {
		w := `(` + TableKeyPairs + `.AppID = ?)`
		wheres = append(wheres, w)
		b = append(b, appID)
	}
	if activeOnly {
		w := `(` + TableLicenses + `.Active = ?)`
		wheres = append(wheres, w)
		b = append(b, activeOnly)
	}

	if len(wheres) > 0 {
		where := " WHERE " + strings.Join(wheres, " AND ")
		q += where
	}

	//complete query
	q += ` ORDER BY ` + TableLicenses + `.ID DESC`
	q += ` LIMIT ` + strconv.FormatInt(limit, 10)

	//run query
	c := sqldb.Connection()
	err = c.SelectContext(ctx, &ll, q, b...)
	if err != nil {
		return
	}

	//Convert datetime to config timezone. This timezone is usually more user
	//friendly than the UTC timezone the datetime is stored as. This conversion
	//isn't handled in SQL since MariaDB and SQLite differ in how they can convert
	//datetimes to different timezones (even though we only support SQLite
	//currently).
	for k, v := range ll {
		if v.DatetimeCreated != "" {
			ll[k].DatetimeCreatedTZ = GetDatetimeInConfigTimezone(v.DatetimeCreated)
		}
	}

	return
}

// GetLicense looks up a single license's data.
func GetLicense(ctx context.Context, licenseID int64, columns sqldb.Columns) (l License, err error) {
	//build base query
	cols, err := columns.ForSelect()
	if err != nil {
		return
	}

	q := `
		SELECT ` + cols + ` 
		FROM ` + TableLicenses + ` 
		LEFT JOIN ` + TableUsers + ` ON ` + TableUsers + `.ID = ` + TableLicenses + `.CreatedByUserID 
		LEFT JOIN ` + TableAPIKeys + ` ON ` + TableAPIKeys + `.ID = ` + TableLicenses + `.CreatedByAPIKeyID 
		JOIN ` + TableKeyPairs + ` ON ` + TableKeyPairs + `.ID = ` + TableLicenses + `.KeyPairID 
		JOIN ` + TableApps + ` ON ` + TableApps + `.ID = ` + TableKeyPairs + `.AppID 
		
		LEFT JOIN ` + TableRenewalRelationships + ` AS rrFrom ON rrFrom.FromLicenseID = ` + TableLicenses + `.ID
		LEFT JOIN ` + TableRenewalRelationships + ` AS rrTo   ON rrTo.ToLicenseID = ` + TableLicenses + `.ID
		
		WHERE ` + TableLicenses + `.ID = ?
	`

	//run query
	c := sqldb.Connection()
	err = c.GetContext(ctx, &l, q, licenseID)
	if err != nil {
		return
	}

	//Convert datetime to config timezone. This timezone is usually more user
	//friendly than the UTC timezone the datetime is stored as. This conversion
	//isn't handled in SQL since MariaDB and SQLite differ in how they can convert
	//datetimes to different timezones (even though we only support SQLite
	//currently).
	if l.DatetimeCreated != "" {
		l.DatetimeCreatedTZ = GetDatetimeInConfigTimezone(l.DatetimeCreated)
		l.IssueDateTZ = GetDateInConfigTimezone(l.IssueDate)
		l.Timezone = config.Data().Timezone
	}

	return
}

// DisableLicense marks a license as inactive. We use a transaction for this
// since we typically will add a note about why the license as disabled as well.
func DisableLicense(ctx context.Context, licenseID int64, tx *sqlx.Tx) (err error) {
	q := `
		UPDATE ` + TableLicenses + `
		SET Active = ?
		WHERE ID = ?
	`
	b := sqldb.Bindvars{
		false,
		licenseID,
	}

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return
	}

	_, err = stmt.ExecContext(ctx, b...)
	return
}
