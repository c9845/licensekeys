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
	"log"
	"time"

	"github.com/c9845/licensekeys/v2/config"
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

// GetDatetimeInConfigTimezone is used to convert a datetime from the database
// as UTC into the timezone specified in the config file.  This cleans up and
// removes duplicate code to do this elsewhere.  This is mostly used for DatetimeCreated
// and DatetimeModified fields.  Although we use time.Parse that can return an
// error, we simply log the error and return the input value instead.  Typically when
// this func is used, the original datetime value is stored in "fieldNameUTC" such as
// DatetimeCreatedUTC for clarification and/or diagnostics.
func GetDatetimeInConfigTimezone(input string) (output string) {
	//the format of datetimes stored in the db
	const format = "2006-01-02 15:04:05"

	//parse datetime into time.Time
	//If we can't parse it, just return the original value.
	t, err := time.Parse(format, input)
	if err != nil {
		log.Println("db.GetDatetimeInConfigTimezone", "could not parse datetime", err)
		output = input
		return
	}

	//get datetime in location specified by timezone from config file
	tInLoc := t.In(config.GetLocation())

	//return string representation of datetime
	output = tInLoc.Format(format)
	return
}

// GetDateInConfigTimezone is similar to GetDatetimeInConfigTimezone but
// for dates only.
func GetDateInConfigTimezone(input string) (output string) {
	//the format of dates stored in the db
	const format = "2006-01-02"

	//parse date into time.Time
	//If we can't parse it, just return the original value.
	t, err := time.Parse(format, input)
	if err != nil {
		log.Println("db.GetDateInConfigTimezone", "could not parse date", err)
		output = input
		return
	}

	//get date in location specified by timezone from config file
	tInLoc := t.In(config.GetLocation())

	//return string representation of date
	output = tInLoc.Format(format)
	return
}
