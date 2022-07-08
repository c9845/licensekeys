# Intro:
This document details how to perform a backup of the app's database when using a SQLite database.


# Notes:
Make sure you test your backups regularly by restoring a backup file as another database file name and checking if all the data is correct. You can manually run queries against the database or stop the app, change the database in the config file, and restart the app. Just make sure no other users are using the app and don't forget to change the database back!

You can also backup a SQLite database by simply copying the database file. However, you need to make sure that the app is not accessing/using the database at the time of the copying due to database locking concerns.


# Basic One-time Backup:
You can backup the database on-demand using the steps below. You would use this prior to installing an update or another maintenance event. Typically scheduled backups, noted below, are more useful.

1. Log on to the server hosting the app (where the database file is located).
1. Stop the app.
    - Run the command `systemctl stop my-app.service` modified as needed for your service's name.
1. Locate the database file. This will be noted in the apps's config file.
1. Make sure the `sqlite3` command line tool is installed. If not, install it.
1. Backup the database.
    - Run the command `sqlite3 my-apps-database.db '.backup my-apps-database.db.bak'` replacing "my-apps-database" with your database file's name.
1. Move the backup file to a secure location.
1. Start the app.
  - Run the command `systemctl start my-app.service` modified as needed for your service's name.
1. Test your backup to make sure it was created correctly.


# Scheduled Backups:
Using scheduled backups helps with taking a snapshot of your database at set times and removes the need for a user to perform backups manually. This ensures you have a copy of the database to fall back upon in case of corruption, user error, or another issue. Typically you would store these backups on another filesystem/server.

To set this up, create a cron task or systemd service & timer to run the above one-time backup commands. Backing up the database may impact performance of the app while the backup is being performed so it is best to perform a backup during off hours.


# Restoring a Backup:
1. You can simply modify the app's config file to point to the database backup file. However, make sure you stop the app's service first.
1. You can also restore to the existing database file using the command `sqlite3 my-apps-database.db '.restore my-apps-database.db.bak'`.
1. Test your restored backup.

