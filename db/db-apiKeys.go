package db

import (
	"context"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/timestamps"
	"github.com/c9845/sqldb/v3"
)

//This table stores API keys used to automate interaction with this app.

// TableAPIKeys is the name of the table
const TableAPIKeys = "api_keys"

// APIKey is used to interact with the table
type APIKey struct {
	ID               int64
	DatetimeCreated  string
	DatetimeModified string
	Active           bool
	CreatedByUserID  int64

	//Key info.
	Description string //so user can identify what the api key is used for
	K           string //the actual api key

	//JOINed fields
	CreatedByUsername string

	//Calculated fields
	DatetimeCreatedInTZ string //DatetimeCreated converted to timezone per config file.
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
func GetAPIKeys(ctx context.Context, activeOnly bool) (aa []APIKey, err error) {
	//Gather columns.
	offset := config.GetTimezoneOffsetForSQLite()
	cols := sqldb.Columns{
		TableAPIKeys + ".*",
		TableUsers + ".Username AS CreatedByUsername",
		`datetime(` + TableAPIKeys + `.DatetimeCreated, '` + offset + `') AS DatetimeCreatedInTZ`,
	}

	colString, err := cols.ForSelect()
	if err != nil {
		return
	}

	//Build query.
	q := `
		SELECT ` + colString + ` 
		FROM ` + TableAPIKeys + `
		JOIN ` + TableUsers + ` ON ` + TableUsers + `.ID=` + TableAPIKeys + `.CreatedByUserID`

	b := sqldb.Bindvars{}
	if activeOnly {
		q += ` WHERE (` + TableAPIKeys + `.Active = ?)`
		b = append(b, true)
	}

	q += ` ORDER BY ` + TableAPIKeys + `.Description`

	//Run query.
	c := sqldb.Connection()
	err = c.SelectContext(ctx, &aa, q, b...)
	return
}

// GetAPIKeyByKey looks up an API key's data by its Key.
func GetAPIKeyByKey(ctx context.Context, key string, columns sqldb.Columns) (a APIKey, err error) {
	cols, err := columns.ForSelect()
	if err != nil {
		return
	}

	q := `
		SELECT ` + cols + `
		FROM ` + TableAPIKeys + `
		WHERE 
			(K = ?)
	`

	c := sqldb.Connection()
	err = c.GetContext(ctx, &a, q, key)
	return
}

// GetAPIKeyByDescription looks up an API key by its Description. This is used when
// adding a new API key to verify a key with the same description doesn't already
// exist and is active.
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

// Insert saves a new API key to the database.
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
	if err != nil {
		return
	}

	a.ID = id
	return
}

// RevokeAPIKey marks an API key as inactive. It cannot be reactivated.
func RevokeAPIKey(ctx context.Context, id int64) error {
	q := `
		UPDATE ` + TableAPIKeys + ` 
		SET 
			DatetimeModified = ?,
			Active = ?
		WHERE 
			(ID = ?)
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

// Update saves changes to an API Key's description or permissions The actual API key
// can never be updated.
func (a *APIKey) Update(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"DatetimeModified",
		"Description",
	}

	colString, err := cols.ForUpdate()
	if err != nil {
		return
	}

	q := `UPDATE ` + TableAPIKeys + ` SET ` + colString + ` WHERE ID = ?`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		timestamps.YMDHMS(),
		a.Description,

		a.ID,
	)
	return
}
