/*
Package db handles interacting with the database.

This file defines the list of queries and funcs that will be used to deploy the database schema.
*/
package db

import "github.com/c9845/sqldb/v2"

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

	//indexes
	//We create indexes in queries separate from CREATE TABLE since these queries are the same accross db
	//types; i.e. we don't need to translate the query based on the db type in use.
	createIndexActivityLogTimestampCreated,
	createIndexActivityLogDatetimeCreated,
	createIndexAPIKeysK,
	createIndexAPIKeysActive,
	createIndexAuthorizedBrowsersRemoteIP,
	createIndexAuthorizedBrowsersCookieUnique,
	createIndexUserLoginsValueUnique,
	createIndexUserLoginsDatetimeCreated,
	createIndexUsersUsername,
	createIndexUsersActive,
	createIndexKeyValueK,
}

var DeployFuncs = []sqldb.DeployFunc{
	insertInitialUser,
	insertInitialAppSettings,

	setLicenseIDStartingValue,
}
