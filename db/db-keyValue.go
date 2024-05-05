package db

import (
	"context"
	"time"

	"github.com/c9845/sqldb/v3"
	"github.com/jmoiron/sqlx"
)

//This table is used for storing random pieces of information.

// TableKeyValue is the name of the table
const TableKeyValue = "key_value"

// KeyValue is used to interact with the table
type KeyValue struct {
	ID               int64
	DatetimeCreated  string
	DatetimeModified string
	Active           bool

	//The actual data being stored
	K          string //key name
	V          string //value
	Type       string //string, float, bool, int, etc. Something to help when converting to datatype for usage. Matches golang type.
	Expiration int64  //a unix timestamp of when key and value should no longer be used, a TTL value. set to 0 for no expiration
}

const (
	createTableKeyValue = `
		CREATE TABLE IF NOT EXISTS ` + TableKeyValue + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated DATETIME DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified DATETIME DEFAULT CURRENT_TIMESTAMP,
			Active INTEGER NOT NULL DEFAULT 1,
			
			K TEXT NOT NULL,
			V TEXT NOT NULL,
			Type TEXT NOT NULL,
			Expiration INTEGER NOT NULL
		)
	`
)

// GetValueByKey looks up key/value pair by the key name.
// This will skip keys that are inactive or expired.
//
// This will skip keys that are inactive or expired. This will return the newest key
// if multiple keys with the same name exist.
func GetValueByKey(ctx context.Context, keyName string) (k KeyValue, err error) {
	//build query
	q := `
		SELECT *
		FROM ` + TableKeyValue + ` 
		WHERE 
			(K = ?)
			AND
			(Active = ?)
			AND
			(Expiration > ? OR Expiration = 0)
		ORDER BY DatetimeCreated DESC
		LIMIT 1
	`
	b := sqldb.Bindvars{
		keyName,
		true, //active
		time.Now().Unix(),
	}

	//run query
	c := sqldb.Connection()
	err = c.GetContext(ctx, &k, q, b...)
	if err != nil {
		return
	}

	return
}

// Insert saves a key/value to the database.
//
// A transaction is optional. It was originally used for when handling database schema
// updates where we save a record if the update was successful to prevent running the
// update multiple times if --update-db is used multiple times.
func (k *KeyValue) Insert(ctx context.Context, tx *sqlx.Tx) (err error) {
	//Use tx if given, otherwise generate a tx.
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

	//Insert into database.
	cols := sqldb.Columns{
		"K",
		"V",
		"Type",
		"Expiration",
	}
	b := sqldb.Bindvars{
		k.K,
		k.V,
		k.Type,
		k.Expiration,
	}
	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableKeyValue + `(` + colString + `) VALUES (` + valString + `)`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, b...)
	if err != nil {
		return
	}

	//Finish tx if we generated it in this func.
	if !txProvided {
		err = tx.Commit()
		if err != nil {
			return
		}
	}

	return
}

// Update saves changes to a key/value.
// Updating is done by key (name), not by ID.
// Don't forget to update expiration if needed!
func (k *KeyValue) Update(ctx context.Context, tx *sqlx.Tx) (err error) {
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

	//build query
	cols := sqldb.Columns{
		"V",
		"Type",
		"Expiration",
		"DatetimeModified",
	}
	colString, err := cols.ForUpdate()
	if err != nil {
		return
	}

	q := `UPDATE ` + TableKeyValue + ` SET ` + colString + ` WHERE ID = ?`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	//run query
	res, err := stmt.ExecContext(
		ctx,

		k.V,
		k.Type,
		k.Expiration,
		k.DatetimeModified,

		k.K,
	)

	id, err := res.LastInsertId()
	if err != nil {
		return
	}
	k.ID = id

	//finish tx if we generated it in this func
	if !txProvided {
		err = tx.Commit()
		if err != nil {
			return
		}
	}

	return
}
