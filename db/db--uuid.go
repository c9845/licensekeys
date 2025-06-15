package db

import (
	"context"
	"errors"

	"github.com/c9845/sqldb/v3"
	"github.com/gofrs/uuid/v5"
)

//This file handles creating UUIDs for the PublicID columns of certain tables. UUIDs
//are used for public-facing API endpoints so that IDs aren't just an incrementing
//integer that is easily guessed.

// UUID represents a UUID, and is stored as a string for use within the app and within
// our database.
type UUID string

// String converts a UUID to a string. We use the UUID type in places for ease of
// identifying what data is stored, but we need a string for interacting with the
// database.
func (u UUID) String() string {
	return string(u)
}

// CreateNewUUID creates a new UUID and makes sure it doesn't already exist in the
// database. This is used for apps, keypairs, and licenses.
//
// The checking for if the UUID is unique is most likely unneccessary, since we would
// have to generate *a lot* of UUIDs for a collision to occur. We just check for
// uniqueness because it is cheap and easy.
//
// We check for if a UUID is unique in all the tables that use UUIDs; we don't want a
// UUID to be duplicated anywhere in the database.
func CreateNewUUID(ctx context.Context) (UUID, error) {
	//Handle collisions by generating UUID multiple times if necessary.
	const attempts = 10
	for i := 0; i < attempts; i++ {
		//Generate the new UUID.
		u, err := uuid.NewV7()
		if err != nil {
			return "", err
		}

		//Convert to string.
		us := u.String()

		//Check if UUID is unique.
		q := `
			SELECT ID FROM ` + TableApps + ` WHERE (PublicID = ?)
			UNION
			SELECT ID FROM ` + TableKeypairs + ` WHERE (PublicID = ?)
			UNION
			SELECT ID FROM ` + TableLicenses + ` WHERE (PublicID = ?)
		`
		b := sqldb.Bindvars{us, us, us}

		c := sqldb.Connection()
		var rows []int64
		err = c.SelectContext(ctx, &rows, q, b...)
		if err != nil {
			return "", err
		}

		//Check if this UUID is already in use.
		if len(rows) == 0 {
			return UUID(us), nil
		}

		//UUID exists, loop and try again.
	}

	//Did not generate a non-unique UUID after trying a bunch of times. Report error.
	return "", errors.New("db.CreateNewUUID: could not create unique UUID")
}
