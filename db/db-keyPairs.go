package db

import (
	"context"
	"database/sql"
	"log"
	"strings"

	"github.com/c9845/licensekeys/licensefile"
	"github.com/c9845/licensekeys/timestamps"
	"github.com/c9845/sqldb/v2"
)

//This table stores the public/private keypair used for generating license signatures.
//
//Each of your apps can have multiple defined sets of key pairs and signing details
//for key rotation purposes or when you want to change the data format or signature
//encoding. Details are not editable once created to prevent issues with already
//generated licenses using the previous values. Instead, create a new set of signing
//details.
//
//The public key is exportable so that you can place it in your application. Private
//key are not exportable for security reasons.

//TODO: should private key be stored encrypted in db? This would require setting a
//password in config file and using it to encrypt/decrypt the private key when it
//is needed. This prevents the private key from being used if the db is stolen/hacked
//but only if the password/config file isn't stolen as well.

//TableKeyPairs is the name of the table.
const TableKeyPairs = "key_pairs"

//KeyPair is used to interact with the table.
type KeyPair struct {
	ID               int64
	DatetimeCreated  string
	DatetimeModified string
	CreatedByUserID  int64
	Active           bool

	AppID               int64                       //what app this key pair is for
	Name                string                      //something descriptive for times when you have multiple key pairs for an app.
	PrivateKey          string                      `json:"-"` //should never leave this app
	PublicKey           string                      //embedded in your application you want to verify the licenses on.
	AlgorithmType       licensefile.KeyPairAlgoType //what algorithm was used to generate the key pair. ex: ecdsa.
	PrivateKeyEncrypted bool                        //true when private key is encrypted with config file encryption key

	//Default sets whether or not this keypair is the default key
	//to use when creating a new license for this app. This keypair
	//will be selected in the select menu automatically when the
	//parent app is chosen for a new license. Only one keypair for
	//an app can be set as default, obviously. If the default keypair
	//is deleted another is not set automatically.
	IsDefault bool
}

const (
	createTableKeyPairs = `
		CREATE TABLE IF NOT EXISTS ` + TableKeyPairs + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified TEXT DEFAULT CURRENT_TIMESTAMP,
			CreatedByUserID INTEGER DEFAULT NULL,
			Active INTEGER NOT NULL DEFAULT 1,

			AppID INTEGER NOT NULL,
			Name TEXT NOT NULL,
			PrivateKey TEXT NOT NULL,
			PublicKey TEXT NOT NULL,
			AlgorithmType TEXT NOT NULL,
			PrivateKeyEncrypted INTEGER NOT NULL,
			IsDefault INTEGER NOT NULL DEFAULT 0,

			FOREIGN KEY (CreatedByUserID) REFERENCES ` + TableUsers + `(ID),
			FOREIGN KEY (AppID) REFERENCES ` + TableApps + `(ID)
		)
	`
)

//GetKeyPairByName looks up a key pair by its name.
func GetKeyPairByName(ctx context.Context, name string) (k KeyPair, err error) {
	q := `
		SELECT ` + TableKeyPairs + `.*
		FROM ` + TableKeyPairs + `
		WHERE Name = ?
	`

	c := sqldb.Connection()
	err = c.GetContext(ctx, &k, q, name)
	return
}

//Validate handles validation of a key pair before saving.
func (k *KeyPair) Validate(ctx context.Context) (errMsg string, err error) {
	//Sanitize.
	k.Name = strings.TrimSpace(k.Name)

	//Validate.
	if k.Name == "" {
		errMsg = "You must provide the name for your key pair."
		return
	}
	if k.AppID < 1 {
		errMsg = "Could not determine which app you are adding a key pair for. Please refresh and try again."
		return
	}

	//Make sure an active keypair with this name doesn't already exist for this app.
	existing, err := GetKeyPairByName(ctx, k.Name)
	if err == sql.ErrNoRows {
		err = nil
	} else if err != nil {
		return
	} else if (err == nil) && (existing.Active) && (existing.AppID == k.AppID) {
		errMsg = "This name is already used for another key pair. Please provide a different name."
		return
	}

	return
}

//Insert saves a key pair.
//You should have already called Validate().
func (k *KeyPair) Insert(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"CreatedByUserID",
		"Active",
		"AppID",
		"Name",
		"PrivateKey",
		"PublicKey",
		"AlgorithmType",
		"PrivateKeyEncrypted",
		"IsDefault",
	}
	b := sqldb.Bindvars{
		k.CreatedByUserID,
		k.Active, //default true
		k.AppID,
		k.Name,
		k.PrivateKey,
		k.PublicKey,
		k.AlgorithmType,
		k.PrivateKeyEncrypted,
		k.IsDefault, //typically false, but will be true if this is the first active keypair for app
	}
	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableKeyPairs + `(` + colString + `) VALUES (` + valString + `)`
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
	k.ID = id
	return
}

//GetPublicKeyByID returns the public key for a key pair. This is used to display
//the public key for copying.
func GetPublicKeyByID(ctx context.Context, id int64) (publicKey string, err error) {
	q := `SELECT PublicKey FROM ` + TableKeyPairs + ` WHERE ID = ?`
	c := sqldb.Connection()
	err = c.GetContext(ctx, &publicKey, q, id)
	return
}

//GetKeyPairs returns the list of key pairs for an app optionally filtered by
//active keypairs only.
func GetKeyPairs(ctx context.Context, appID int64, activeOnly bool) (kk []KeyPair, err error) {
	//Base query.
	q := `
		SELECT ` + TableKeyPairs + `.* 
		FROM ` + TableKeyPairs

	//Build WHEREs
	wheres := []string{}
	b := sqldb.Bindvars{}

	w := `(` + TableKeyPairs + `.AppID = ?)`
	wheres = append(wheres, w)
	b = append(b, appID)

	if activeOnly {
		w := `(` + TableKeyPairs + `.Active = ?)`
		wheres = append(wheres, w)
		b = append(b, activeOnly)
	}

	if len(wheres) > 0 {
		where := " WHERE " + strings.Join(wheres, " AND ")
		q += where
	}

	//Complete query.
	q += ` ORDER BY ` + TableKeyPairs + `.Active DESC, ` + TableKeyPairs + `.Name ASC`

	//Run query.
	c := sqldb.Connection()
	err = c.SelectContext(ctx, &kk, q, b...)
	return
}

//Delete marks a defined custom field as deleted.
func (k *KeyPair) Delete(ctx context.Context) (err error) {
	q := `
		UPDATE ` + TableKeyPairs + ` 
		SET 
			Active = ?,
			DatetimeModified = ?,
			IsDefault = ?
		WHERE ID = ?
	`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}

	_, err = stmt.ExecContext(
		ctx,

		false,
		timestamps.YMDHMS(),
		false, //set Default to false just to keep only one keypair as default=true max

		k.ID,
	)
	return
}

//GetKeyPairByID looks up a key pair by its ID.
func GetKeyPairByID(ctx context.Context, id int64) (k KeyPair, err error) {
	q := `
		SELECT ` + TableKeyPairs + `.*
		FROM ` + TableKeyPairs + `
		WHERE ID = ?
	`

	c := sqldb.Connection()
	err = c.GetContext(ctx, &k, q, id)
	return
}

//SetIsDefault marks the keypair as the default keypair for the respective app. This
//also marks any other keypairs as non-default for the app to ensure only one
//keypair is marked as default as a time.
func (kp *KeyPair) SetIsDefault(ctx context.Context) (err error) {
	//look up details of key pair to get app id to set other
	//keypairs as inactive.
	*kp, err = GetKeyPairByID(ctx, kp.ID)
	if err != nil {
		return
	}

	//use transaction since we are doing multiple queries
	c := sqldb.Connection()
	tx, err := c.BeginTxx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	//set all keypairs for this app as non default
	q := `
		UPDATE ` + TableKeyPairs + `
		SET IsDefault = ?
		WHERE AppID = ?
	`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		log.Println(q, kp.AppID)
		return
	}
	_, err = stmt.ExecContext(
		ctx,
		false, //Default
		kp.AppID,
	)
	if err != nil {
		return
	}

	//mark this keypair as the default
	q = `
		UPDATE ` + TableKeyPairs + ` 
		SET IsDefault = ?
		WHERE ID = ?
	`
	stmt, err = tx.PrepareContext(ctx, q)
	if err != nil {
		log.Println(q)
		return
	}
	_, err = stmt.ExecContext(
		ctx,
		true, //Default
		kp.ID,
	)
	if err != nil {
		return
	}

	//commit
	err = tx.Commit()
	return
}

//GetDefaultKeyPair retuns the default key pair for an app.
func GetDefaultKeyPair(ctx context.Context, appID int64) (k KeyPair, err error) {
	//Base query.
	q := `
		SELECT ` + TableKeyPairs + `.* 
		FROM ` + TableKeyPairs + `
		WHERE
			(` + TableKeyPairs + `.AppID = ?)
			AND
			(` + TableKeyPairs + `.IsDefault = ?)
	`

	//rRun query.
	c := sqldb.Connection()
	err = c.GetContext(ctx, &k, q, appID, true)
	return
}
