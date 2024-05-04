package db

import "github.com/c9845/sqldb/v3"

// DeployQueries is the list of queries to deploy the database schema. This only includes
// CREATE TABLE queries.
// - Inserting of initial data is handled via DeployFuncs (see below).
// - Queries to create indexes are located in createIndexes (see db--indexes.go).
var DeployQueries = []string{
	createTableKeyValue,

	createTableUsers,
	createTableAuthorizedBrowsers,
	createTableUserLogins,

	createTableAPIKeys,
	createTableActivityLog,
	createTableAppSettings,

	createTableApps,
	createTableKeyPairs,
	createTableCustomFieldsDefined,
	createTableCustomFieldResults,
	createTableLicenses,
	createTableDownloadHistory,
	createTableLicenseNotes,
	createTableRenewalRelationships,
}

var DeployFuncs = []sqldb.QueryFunc{
	insertInitialUser,
	insertInitialAppSettings,

	setLicenseIDStartingValue,
}
