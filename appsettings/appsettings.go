/*
Package appsettings handles settings that change the functionality of this app. These
settings turn certain features or checks on or off or otherwise alter the way the app
looks or works. Most of the time app settings should be "opt in".

How to add a new app setting:
  - Define a name for the setting, somthing short but descriptive, in camelcase.
  - Name should be such that the "false" state is default.
  - Add the setting to the AppSetting struct (db-appsettings.go).
  - Update the createTableAppSettings function with the new column (db-appsettings.go).
  - Update the updateTableAppSettings function with the new column (db-appsettings.go).
  - Update the insertIntialAppSettings function with the new column (db-appsettings.go).
  - Update the Update function with the new column (db-appsettings.go).
  - Update the appsettings type (types.ts).
  - Document new app setting in help files.
  - Deploy the db or update the db schema as needed.
*/
package appsettings

import (
	"encoding/json"
	"net/http"

	"github.com/c9845/licensekeys/v4/db"
	"github.com/c9845/output"
)

// Get looks up the app settings.
func Get(w http.ResponseWriter, r *http.Request) {
	data, err := db.GetAppSettings(r.Context())
	if err != nil {
		output.Error(err, "Could not look up app settings.", w)
		return
	}

	output.DataFound(data, w)
}

// Update saves changes to app settings. All settings are updated.
func Update(w http.ResponseWriter, r *http.Request) {
	//Get input data.
	raw := r.FormValue("data")

	//Parse data into struct.
	var ad db.AppSettings
	err := json.Unmarshal([]byte(raw), &ad)
	if err != nil {
		output.Error(err, "Could not parse data to update app settings.", w)
		return
	}

	//Update db.
	//At least one administrator user must have 2FA turned on prior to forcing 2FA
	//for all users since if no user has 2FA enabled you would be locked out of the
	//app.
	err = ad.Update(r.Context())
	if err == db.ErrNoUser2FAEnabled {
		output.Error(err, "No administrator user has 2 Factor Authentication enabled yet. You must have at least one administrator with 2FA turned on before forcing all users to use 2FA.", w)
		return
	} else if err != nil {
		output.Error(err, "Could not update app settings.", w)
		return
	}

	output.UpdateOK(w)
}
