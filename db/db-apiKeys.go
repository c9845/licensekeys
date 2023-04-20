package db

import (
	"context"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/timestamps"
	"github.com/c9845/sqldb/v2"
)

//This table stores API keys used to automate interaction with this app.

// TableAPIKeys is the name of the table
const TableAPIKeys = "api_keys"

// APIKey is used to interact with the table
type APIKey struct {
	ID               int64
	DatetimeCreated  string
	DatetimeModified string
	CreatedByUserID  int64
	Active           bool

	Description string //so user can identify what the api key is used for
	K           string //the actual api key

	//JOINed fields
	CreatedByUsername string

	//Calculated fields
	DatetimeCreatedTZ string //DatetimeCreated converted to timezone per config file.
	Timezone          string //extra data for above fields for displaying in GUI.
}

const (
	createTableAPIKeys = `
		CREATE TABLE IF NOT EXISTS ` + TableAPIKeys + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified TEXT DEFAULT CURRENT_TIMESTAMP,
			Active INTEGER NOT NULL DEFAULT 1,
			CreatedByUserID INTEGER NOT NULL,
			
			Description TEXT NOT NULL,
			K TEXT NOT NULL,

			FOREIGN KEY(CreatedByUserID) REFERENCES ` + TableUsers + `(ID)
		)
	`

	createIndexAPIKeysK      = `CREATE UNIQUE INDEX IF NOT EXISTS ` + TableAPIKeys + `__K_idx ON ` + TableAPIKeys + ` (K)`
	createIndexAPIKeysActive = `CREATE INDEX IF NOT EXISTS ` + TableAPIKeys + `__Active_idx ON ` + TableAPIKeys + ` (Active)`
)

// GetAPIKeys looks up a list of API keys.
func GetAPIKeys(ctx context.Context, activeOnly bool, columns sqldb.Columns) (aa []APIKey, err error) {
	//build columns
	cols, err := columns.ForSelect()
	if err != nil {
		return
	}

	//build query
	q := `
		SELECT ` + cols + ` 
		FROM ` + TableAPIKeys + `
		JOIN ` + TableUsers + ` ON ` + TableUsers + `.ID=` + TableAPIKeys + `.CreatedByUserID`

	b := sqldb.Bindvars{}
	if activeOnly {
		q += ` WHERE (` + TableAPIKeys + `.Active = ?)`
		b = append(b, true)
	}

	q += ` ORDER BY ` + TableAPIKeys + `.Description`

	//run query
	c := sqldb.Connection()
	err = c.SelectContext(ctx, &aa, q, b...)

	//handle converting datetimes to correct timezone
	//This isn't handled in sql query since mariadb and sqlite differ in how they can
	//convert a datetime to a different timezone.  Doing it in this manner ensures the
	//same conversion method is applied so golang does the conversion.
	for k, v := range aa {
		aa[k].DatetimeCreatedTZ = GetDatetimeInConfigTimezone(v.DatetimeCreated)
		aa[k].Timezone = config.Data().Timezone
	}

	return
}

// GetAPIKeyByKey looks up an api key by its Key.
func GetAPIKeyByKey(ctx context.Context, key string, columns sqldb.Columns) (a APIKey, err error) {
	cols, err := columns.ForSelect()
	if err != nil {
		return
	}

	q := `
		SELECT ` + cols + `
		FROM ` + TableAPIKeys + `
		WHERE K=?
	`

	c := sqldb.Connection()
	err = c.GetContext(ctx, &a, q, key)
	return
}

// GetAPIKeyByDescription looks up an API key by its Description
// This is used when adding an API key to verify that a description isn't used
// for more than one active key. We don't want duplicate descriptions to reduce
// confusion and mistakes with multiple active API keys for the same usage.
func GetAPIKeyByDescription(ctx context.Context, desc string) (a APIKey, err error) {
	q := `
		SELECT ` + TableAPIKeys + `.* 
		FROM ` + TableAPIKeys + `
		WHERE 
			(` + TableAPIKeys + `.Description = ?)
			AND
			(` + TableAPIKeys + `.Active = ?)
	`

	c := sqldb.Connection()
	err = c.GetContext(ctx, &a, q, desc, true)
	return
}

// Insert saves a new api to the database.
// you should have already performed validation.
func (a *APIKey) Insert(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"CreatedByUserID",
		"Description",
		"K",
	}
	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableAPIKeys + `(` + colString + `) VALUES (` + valString + `)`

	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(
		ctx,

		a.CreatedByUserID,
		a.Description,
		a.K,
	)
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	a.ID = id
	return
}

// RevokeAPIKey marks an api key as inactive
func RevokeAPIKey(ctx context.Context, id int64) error {
	q := `
		UPDATE ` + TableAPIKeys + ` 
		SET 
			DatetimeModified = ?,
			Active = ?
		WHERE ID = ?
	`

	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		timestamps.YMDHMS(),
		false,
		id,
	)
	return err
}
