package db

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/sqldb/v3"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v3"
)

//This table keeps a record of each license that was generated and the common data
//for each license.
//
// A license is created once, and the signature is generated once, so we need to keep
// a copy of each piece of data included in a license so that a license can be
// downloaded at any time in the future. The signature is NOT recalculated each time
// a license is downloaded to (1) prevent extra work, and (2) because it should not
// be changing.
//
// You would use the data in this table in conjunction with the custom_field_results
// table to "rebuild" a license to allow for redownloading it.

// TableLicenses is the name of the table.
const TableLicenses = "licenses"

// License is used to interact with the table.
type License struct {
	ID               int64
	PublicID         UUID //Used when interacting with the public API only; mainly so public-facing ID isn't just an incrementing number.
	DatetimeCreated  string
	DatetimeModified string
	Active           bool

	//A license can be created by a user or via an API call.
	CreatedByUserID   null.Int
	CreatedByAPIKeyID null.Int

	//Details chosen/provided in GUI by user, or API call.
	AppID          int64  //the app this license if for. can also get this through keypair.
	KeyPairID      int64  //the keypair used to sign the license
	CompanyName    string //company this license is for
	ContactName    string //who requested this license
	PhoneNumber    string //contact info
	Email          string //contact info
	IssueDate      string //yyyy-mm-dd, set by server, UTC timezone.
	IssueTimestamp int64  //unix timestamp in seconds
	ExpireDate     string //yyyy-mm-dd, set by input type=date in GUI so timezone is dependent on user's location.

	//Fingerprint is a hash of a license file's data, and is what is signed with the
	//private key to generate the signature.
	Fingerprint string

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

	//Fields copied from app when license is created. This data has to be copied
	//because it could be used/included in a license file, and therefore a signature
	//would be based off of its presence. If the settings for the app changes, we
	//don't want license to become invalid or signature to change.
	AppName       string
	ShowLicenseID bool
	ShowAppName   bool
	FileFormat    string

	//Calculated fields
	Expired             bool   //true if Expire date is greater than current date
	DatetimeCreatedInTZ string //DatetimeCreated converted to timezone per config file.
	IssueDateInTZ       string // " " " "
	Timezone            string //extra data for above fields for displaying in GUI.

	//JOINed fields
	KeypairAlgo                string
	FingerprintAlgo            string
	EncodingAlgo               string
	CreatedByUsername          null.String
	CreatedByAPIKeyDescription null.String
	AppDownloadFilename        string
	RenewedFromLicenseID       null.Int
	RenewedToLicenseID         null.Int
}

const (
	createTableLicenses = `
		CREATE TABLE IF NOT EXISTS ` + TableLicenses + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			PublicID TEXT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified TEXT DEFAULT CURRENT_TIMESTAMP,
			Active INTEGER NOT NULL DEFAULT 1,
			
			CreatedByUserID INTEGER DEFAULT NULL,
			CreatedByAPIKeyID INTEGER DEFAULT NULL,

			AppID INTEGER NOT NULL,
			KeyPairID INTEGER NOT NULL,
			CompanyName TEXT NOT NULL,
			ContactName TEXT NOT NULL,
			PhoneNumber TEXT NOT NULL,
			Email TEXT NOT NULL,
			IssueDate TEXT NOT NULL,
			IssueTimestamp INT NOT NULL,
			ExpireDate TEXT NOT NULL,

			Fingerprint TEXT NOT NULL,
			Signature TEXT NOT NULL,
			Verified INTEGER NOT NULL DEFAULT 0,

			AppName TEXT NOT NULL,
			ShowLicenseID INTEGER NOT NULL DEFAULT 0,
			ShowAppName INTEGER NOT NULL DEFAULT 0,
			FileFormat TEXT NOT NULL,

			FOREIGN KEY (CreatedByUserID) REFERENCES ` + TableUsers + `(ID),
			FOREIGN KEY (CreatedByAPIKeyID) REFERENCES ` + TableAPIKeys + `(ID),
			FOREIGN KEY (KeyPairID) REFERENCES ` + TableKeypairs + `(ID)
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

	now := time.Now().UTC() //Need UTC since time.Parse() is using UTC.
	if !expDate.After(now) {
		log.Println("license.Validate()", "expireDate", expDate)
		log.Println("license.Validate()", "now       ", now)

		errMsg = "The expiration date must be in the future."
		return
	}

	return
}

// Insert saves a license. You should have already called Validate().
//
// After Insert() is called, you still need to validate the license and update the
// Signature and Verified fields. These fields are set to blank/false to prevent a
// license from being used until after it has been verified. Verification performs a
// check to make sure a license is actually usable by a 3rd party.
func (l *License) Insert(ctx context.Context, tx *sqlx.Tx) (err error) {
	if l.CreatedByUserID.Int64 == 0 && l.CreatedByAPIKeyID.Int64 == 0 {
		return errors.New("cannot determine how license is being added")
	}

	uuid, err := CreateNewUUID(ctx)
	if err != nil {
		return
	}
	l.PublicID = uuid

	cols := sqldb.Columns{
		"PublicID",
		"DatetimeCreated",
		"Active",

		"AppID",
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
		"ShowLicenseID",
		"ShowAppName",
		"FileFormat",
	}
	b := sqldb.Bindvars{
		l.PublicID,
		l.DatetimeCreated,
		true, //Active

		l.AppID,
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
		l.ShowLicenseID,
		l.ShowAppName,
		l.FileFormat,
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
	defer stmt.Close()

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
		SET 
			Fingerprint = ?,
			Signature = ?
		WHERE 
			(ID = ?)
	`

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, l.Fingerprint, l.Signature, l.ID)
	return
}

// MarkVerified updates a saved license by marking it as valid.
//
// This is done after a license is created and saved to the database, but before a
// license is available to download/view. After creating a license, it is marked as
// invalid. Then, we "read" the license like a 3rd-party app would, and verify it with
// the public key. This ensures that the created license is actually usable. After
// this process is done, the license is marked as verified.
func (l *License) MarkVerified(ctx context.Context) (err error) {
	q := `
		UPDATE ` + TableLicenses + ` 
		SET Verified = ?
		WHERE 
			(ID = ?)
	`

	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, l.Verified, l.ID)
	return
}

// GetLicenses looks up a list of licenses optionally filtered by app and active
// licenses only.
func GetLicenses(ctx context.Context, appID, limit int64, activeOnly bool, columns sqldb.Columns) (ll []License, err error) {
	//Build query.
	cols, err := columns.ForSelect()
	if err != nil {
		return
	}

	q := `
		SELECT ` + cols + ` 
		FROM ` + TableLicenses + ` 
		JOIN ` + TableKeypairs + ` ON ` + TableKeypairs + `.ID=` + TableLicenses + `.KeyPairID 
		JOIN ` + TableApps + ` ON ` + TableApps + `.ID=` + TableKeypairs + `.AppID

		LEFT JOIN ` + TableRenewalRelationships + ` AS rrFrom ON rrFrom.FromLicenseID = ` + TableLicenses + `.ID
		LEFT JOIN ` + TableRenewalRelationships + ` AS rrTo   ON rrTo.ToLicenseID = ` + TableLicenses + `.ID
	`

	wheres := []string{}
	b := sqldb.Bindvars{}
	if appID > 0 {
		w := `(` + TableKeypairs + `.AppID = ?)`
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

	q += ` ORDER BY ` + TableLicenses + `.ID DESC`
	q += ` LIMIT ` + strconv.FormatInt(limit, 10)

	//Run query.
	c := sqldb.Connection()
	err = c.SelectContext(ctx, &ll, q, b...)
	if err != nil {
		return
	}

	return
}

// GetLicense looks up a single license's data.
func GetLicense(ctx context.Context, licenseID int64, columns sqldb.Columns) (l License, err error) {
	//Build query.
	cols, err := columns.ForSelect()
	if err != nil {
		return
	}

	q := `
		SELECT ` + cols + ` 
		FROM ` + TableLicenses + ` 
		LEFT JOIN ` + TableUsers + ` ON ` + TableUsers + `.ID = ` + TableLicenses + `.CreatedByUserID 
		LEFT JOIN ` + TableAPIKeys + ` ON ` + TableAPIKeys + `.ID = ` + TableLicenses + `.CreatedByAPIKeyID 
		JOIN ` + TableKeypairs + ` ON ` + TableKeypairs + `.ID = ` + TableLicenses + `.KeyPairID 
		JOIN ` + TableApps + ` ON ` + TableApps + `.ID = ` + TableKeypairs + `.AppID 
		
		LEFT JOIN ` + TableRenewalRelationships + ` AS rrFrom ON rrFrom.FromLicenseID = ` + TableLicenses + `.ID
		LEFT JOIN ` + TableRenewalRelationships + ` AS rrTo   ON rrTo.ToLicenseID = ` + TableLicenses + `.ID
		
		WHERE ` + TableLicenses + `.ID = ?
	`

	//Run query.
	c := sqldb.Connection()
	err = c.GetContext(ctx, &l, q, licenseID)
	if err != nil {
		return
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
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, b...)
	return
}
