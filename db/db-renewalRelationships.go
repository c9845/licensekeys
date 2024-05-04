package db

import (
	"context"

	"github.com/c9845/sqldb/v3"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v3"
)

//This table keeps a record of the relationship that arises from renewing a
//license. This stores the "from" to "to" relationship.

// TableRenewalRelationships is the name of the table.
const TableRenewalRelationships = "renewal_relationships"

// RenewalRelationship is used to interact with the table.
type RenewalRelationship struct {
	ID               int64
	DatetimeCreated  string
	DatetimeModified string
	Active           bool

	//a license can be created by a user or via an api call.
	CreatedByUserID   null.Int
	CreatedByAPIKeyID null.Int

	FromLicenseID int64
	ToLicenseID   int64
}

const (
	createTableRenewalRelationships = `
		CREATE TABLE IF NOT EXISTS ` + TableRenewalRelationships + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified TEXT DEFAULT CURRENT_TIMESTAMP,
			Active INTEGER NOT NULL DEFAULT 1,

			CreatedByUserID INTEGER DEFAULT NULL,
			CreatedByAPIKeyID INTEGER DEFAULT NULL,
			
			FromLicenseID INTEGER NOT NULL,
			ToLicenseID INTEGER NOT NULL,

			FOREIGN KEY (CreatedByUserID) REFERENCES ` + TableUsers + `(ID),
			FOREIGN KEY (CreatedByAPIKeyID) REFERENCES ` + TableAPIKeys + `(ID),
			FOREIGN KEY (FromLicenseID) REFERENCES ` + TableLicenses + `(ID),
			FOREIGN KEY (ToLicenseID) REFERENCES ` + TableLicenses + `(ID)
		)
	`
)

// Insert saves a new renewal relationship.
func (r *RenewalRelationship) Insert(ctx context.Context, tx *sqlx.Tx) (err error) {
	cols := sqldb.Columns{
		"DatetimeCreated",
		"FromLicenseID",
		"ToLicenseID",
	}
	b := sqldb.Bindvars{
		r.DatetimeCreated,
		r.FromLicenseID,
		r.ToLicenseID,
	}

	if r.CreatedByUserID.Int64 > 0 {
		cols = append(cols, "CreatedByUserID")
		b = append(b, r.CreatedByUserID.Int64)
	} else {
		cols = append(cols, "CreatedByAPIKeyID")
		b = append(b, r.CreatedByAPIKeyID.Int64)
	}

	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableRenewalRelationships + `(` + colString + `) VALUES ( ` + valString + `)`
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
	r.ID = id
	return
}

// GetRenewalRelationshipByFromID looks up a renewal relationship's data by the
// from license's ID.
func GetRenewalRelationshipByFromID(ctx context.Context, fromLicenseID int64) (r RenewalRelationship, err error) {
	q := `
		SELECT 
			` + TableRenewalRelationships + `.*
		FROM ` + TableRenewalRelationships + ` 
		WHERE (` + TableRenewalRelationships + `.FromLicenseID = ?)
	`
	c := sqldb.Connection()
	err = c.GetContext(ctx, &r, q, fromLicenseID)
	return
}
