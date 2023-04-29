# Intro:
This document details the process of installing an update for the License Key Server application.


# Basics:
Updates are provided via the same method as when you first installed the app: the app's files are simply provided via a zip file. You need to unzip the contents of the zip file, copy/move and overwrite the old versions of the files, and apply any database schema updates. The update process should not take more than a few minutes in most circumstances. The changelog with each version will note if a database schema update is required and what changes have been made.


# Warnings:
- You should backup of the apps's database prior to installing an update.  
- You should take a backup of the server the app runs on prior to installing an update.
- You must stop the app's service prior to installing an update to ensure there are no database contention issues that could possibly corrupt the database.
- You should read the changelog in the update to understand the changes in the new version.
- If you modified any files within the app's directory, these files will be overwritten. Copy and backup any files with changes to a different directory.


# Install an Update:
1. Back up your system and the app's database.
    - Take a snapshot of the server.
    - Use SQLite's built in backup command.
    - Create a copy of the app's files.
    - Use another backup tool.
    - Verify you can restore from the backup!

1. Make note of the old version of the app you are currently using in case you need to revert to an older version.
    - Open the "VERSION.txt" file. 
    - Run the binary with the `--version` flag.
    - Check the app's diagnostic page.

1. Stop the app from running.
    - We don't want to corrupt the app's database.
    - `systemctl stop my-app.service`.

1. Get the updated app files.
    - Get files from Github or elsewhere. Files should be downloaded as a zip archive and a checksum. 
    - Confirm the checksum matches for the zip file.
        - For Ubuntu or other Linux systems, use the command `sha256sum`.
        - For Windows, use the powershell command `Get-FileHash`.
    - Unzip the zip file.

1. Update the app.
    - Copy/move the unzip files and overwrite the existing files.
    - If the changelog shows the database schema has changed, run the binary using the `--deploy-db` and/or `--update-db` flag as needed.

1. Start the app manually to check for startup issues.
    - If you see any errors, identify and fix the issues.
    - Exit the manually run version of the app.

1. Start the app's service and use it like normal.
    - For example: `systemctl start my-app.service`.

1. Monitor the logs for any issues.
