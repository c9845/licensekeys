/*
Package db provides functions to interact with a sql database.

This package does not handle db connection. For that, refer to the 3rd part sqldb
package. This package handles higher-level stuff specific to this app, not low
level db configuration or connection.

Other files in this package declare the schema for each table. Each table should
be in its own file named as close as possible to the table name. While table names
are spaced with underscores these files are named with camelcase. Each file defines
the struct that is used to interface with that table's data. Struct names should be
similar to table names except in camelcase and in singular form (tables are plural).
The fields in each struct should match exactly to column names. Fields are grouped
into three main sections: table columns, calculated fields (stuff calculated from
other columns or other data), and JOINed fields (columns aliased from other tables
in JOIN queries).

Each file should declare a "create" query that deployes the schema for this table.
There can also be additional queries to be run after the table is created that creates
indexes and/or inserts initial data into the database. These queries should all be
capable of rerunning safely (i.e. not creating duplicate tables, duplicate indexes,
or inserting default data multiple times).

Each file may have an "update" query that adds new columns to the table. Any new
columns should be added to the "create" and "update" funcs to handle times when
user is installing app fresh or upgrading an existing install with existing data
in the database.
*/
package db

import (
	"errors"
)

// errors
var (
	//ErrInputInvalid is returned when doing validation before INSERT or UPDATE
	//queries and some input value is not valid.
	ErrInputInvalid = errors.New("db: input invalid")

	//ErrCouldNotFindMatchingField is returned when validating the list of defined
	//versus result fields and a match could not be found. This means that the
	//user somehow provided an incorrect result.
	ErrCouldNotFindMatchingField = errors.New("could not find matching result for defined field")
)
