# Intro:
This document details the process of installing a software update for the app.


# Warnings:
- You should backup of the apps's database prior to installing an update.  
- You should take a backup of the server the app runs on prior to installing an update.
- You must stop the app's service prior to installing an update to ensure there are no database contention issues that could possibly corrupt the database.
- You should read the changelog in the update to understand the changes in the new version.
- If you modified any files within the app's directory, these files will be overwritten. Copy and backup any files with changes to a different directory.


# Basics:
Updates are provided via the same method as when you first installed the app: the app's files are simply provided via a zip file. You need to extract the contents of the zip file, copy/move and overwrite the old versions of the files, and apply any database schema updates. The update process should not take more than a few minutes in most circumstances. The changelog with each version will note if a database schema update is required and what changes have been made.


# Install an Update:
1. Get the updated app files.
    - Get files from Github or elsewhere. Files should be downloaded as a zip archive and a checksum. 
    - Confirm the checksum matches for the zip file.
        - For Ubuntu or other Linux systems, use the command `sha256sum`.
        - For Windows, use the powershell command `Get-FileHash`.
    - Extract the zip file.
    - Make note of the old version of the app you are currently using. Open the "VERSION.txt" file or run the app's binary with the `--version` flag.

1. Make sure you have a backup of your database.

1. Make sure you have a backup of the server the app is hosted on.

1. Stop the app.
    - `systemctl stop my-app.service`.

1. Copy the new files and overwrite the old files for the app.
    - App's binary.
    - Any files in other directories (website, documentation, etc.).
    - LICENSE, COPYRIGHT, README.

1. Read the changelog to see what has changed in this version of the app.
    - The changelog will note if database changes were made.
    - The changelog will tell you of any breaking changes, new functionality, or bug fixes.

1. Update the app's database schema.
    - This is only required if you see schema changes noted in the changelog.
    - Run the binary with the `--update-db` flag.

1. Start the app manually to check for startup issues.
    - If you see any errors, identify and fix the issues.
    - Exit the manually run version of the app.

1. Start the app's service and use it like normal.
    - For example: `systemctl start my-app.service`.

1. Monitor the logs for any issues.

1. It is advised to keep your backups for as long as possible in case any issues arise. It is also advised to save at least one old version of the app.
