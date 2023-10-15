package db

func init() {
	DeployQueries = append(DeployQueries, createIndexes...)
	UpdateQueries = append(UpdateQueries, createIndexes...)
}

var createIndexes = []string{
	//Index are separate queries from CREATE TABLE since the queries we use to create
	//indexes work for both MariaDB and SQLite. When creating indexes as part of CREATE
	//TABLE queries there are slight differences in how the queries are written for
	//MariaDB versus SQLite. This just allows for ease of using both database types
	//and not needing to translate the index creation part of a CREATE TABLE query.

	//We store the indexes in a separate var, not part of DeployQueryies or UpdateQueries
	//since this way we only have to update, and keep track of, one place that defines
	//all the queries that will run to create indexes on the database regardless of
	//deploying or updating. These queries will be appended to the DeployQueries and
	//UpdateQueries as noted in init() above. Basically, this prevents us from adding
	//a new index upon deploy but forgetting it for updating a schema, or vice-versa.

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
}
