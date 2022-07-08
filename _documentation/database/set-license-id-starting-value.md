# Intro:
This document details setting the starting, or next, ID for created licenses.


# Basics:
The app initially deploys with license IDs starting at 10,000. You can alter the starting value to any number when the database is first deployed. If at least one license has been created you can set the next ID to be used but it must be larger than any existing ID.

You cannot change the starting, or next value, from within the app. However, you can change it directly in the database.


# Updating the Database:
This assumes you have the `sqlite3` command line tool installed on the system where the app's database file is stored.

Get the current max ID from the license table using the following queries:
  - `SELECT seq FROM SQLITE_SEQUENCE WHERE name = 'licenses';`
  - `SELECT MAX(ID) FROM licenses;`
 
These two queries should return the same value. You cannot set the next ID value below this number. To set the next ID to increment from, run one of the following queries based on if you have created any license:
 - When no license exist: `INSERT INTO SQLITE_SEQUENCE (name, seq) VALUES ('licenses', ?);` where `?` is the value you wish to start at in a new, clean database.
 - When at least once license has been created: `UPDATE SQLITE_SEQUENCE SET seq = ? WHERE name = 'licenses';` where `?` is the value higher than the values you got in the above queries for a database with existing data.
