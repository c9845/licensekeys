package license

import (
	"encoding/json"
	"net/http"

	"github.com/c9845/licensekeys/v4/db"
	"github.com/c9845/licensekeys/v4/users"
	"github.com/c9845/output"
	"gopkg.in/guregu/null.v3"
)

// CreateViaGUI creates a license via the app's GUI.
func CreateViaGUI(w http.ResponseWriter, r *http.Request) {
	//Parse and validate basic license file data from request.
	//
	//Via the GUI, an app and keypair are chosen and data for both are provided here.
	rawLicenseData := r.FormValue("licenseData")
	var l db.License
	err := json.Unmarshal([]byte(rawLicenseData), &l)
	if err != nil {
		output.Error(err, "Could not parse license data to create license.", w)
		return
	}

	errMsg, err := l.Validate(r.Context())
	if err != nil && errMsg != "" {
		output.Error(err, errMsg, w)
		return
	} else if err != nil {
		output.Error(err, "Could not validate data about this license.", w)
		return
	} else if errMsg != "" {
		output.ErrorInputInvalid(errMsg, w)
		return
	}

	//Look up and validate app.
	app, err := db.GetAppByID(r.Context(), l.AppID)
	if err != nil {
		output.Error(err, "Could not look up app.", w)
		return
	}
	if !app.Active {
		output.ErrorInputInvalid("This app is not active, you cannot create licenses for it.", w)
		return
	}

	//Look up and validate keypair.
	if l.KeyPairID < 1 {
		//Look up default keypair for app. This should never happen since a user has
		//to pick a keypair in the GUI.
		defaultKP, err := db.GetDefaultKeypairForAppID(r.Context(), l.AppID)
		if err != nil {
			output.Error(err, "Could not look up default keypair for app.", w)
			return
		}
		l.KeyPairID = defaultKP.ID
	}

	kp, err := db.GetKeypairByID(r.Context(), l.KeyPairID)
	if err != nil {
		output.Error(err, "Could not look up keypair.", w)
		return
	}
	if !kp.Active {
		output.ErrorInputInvalid("This keypair is not active, you cannot create licenses with it.", w)
		return
	}

	//Parse and validate custom fields.
	rawCustomFields := r.FormValue("customFields")
	var cfr db.MultiCustomFieldResult
	err = json.Unmarshal([]byte(rawCustomFields), &cfr)
	if err != nil {
		output.Error(err, "Could not parse custom fields to create license.", w)
		return
	}

	errMsg, err = cfr.Validate(r.Context(), l.AppID, false)
	if err != nil && errMsg != "" {
		output.Error(err, errMsg, w)
		return
	} else if err != nil {
		output.Error(err, "Could not validate custom fields to create license.", w)
		return
	} else if errMsg != "" {
		output.ErrorInputInvalid(errMsg, w)
		return
	}

	//Get user who is creating this license.
	loggedInUserID, err := users.GetUserIDFromRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	l.CreatedByUserID = null.IntFrom(loggedInUserID)

	//Create and save license.
	licenseID, _, err := create(r.Context(), l, cfr, app, kp, 0)
	if err != nil {
		output.Error(err, "Could not create license.", w)
		return
	}

	output.InsertOK(licenseID, w)
}
