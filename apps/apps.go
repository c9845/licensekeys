/*
Package apps handles the apps that you want to create license keys for. This adds,
edits, and returns the list of apps defined.

This does not refer to "this" license key server app.
*/
package apps

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/users"
	"github.com/c9845/output"
)

//Add saves a new app.
func Add(w http.ResponseWriter, r *http.Request) {
	//Get input data.
	raw := r.FormValue("data")

	//Parse data into struct.
	var a db.App
	err := json.Unmarshal([]byte(raw), &a)
	if err != nil {
		output.Error(err, "Could not parse data to add app.", w)
		return
	}

	//Make sure this isn't being called with an already existing app.
	if a.ID != 0 {
		output.ErrorAlreadyExists("Could not determine if you are adding or updating an app.", w)
		return
	}

	//Validate..
	errMsg, err := a.Validate(r.Context())
	if err != nil && errMsg != "" {
		output.Error(err, errMsg, w)
		return
	} else if err != nil {
		output.Error(err, "Could not validate data about this app.", w)
		return
	} else if errMsg != "" {
		output.ErrorInputInvalid(errMsg, w)
		return
	}

	//Get user who is adding this app.
	loggedInUserID, err := users.GetUserIDByRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	a.CreatedByUserID = loggedInUserID

	//Save.
	err = a.Insert(r.Context())
	if err != nil {
		output.Error(err, "Could not save app.", w)
		return
	}

	output.InsertOK(a.ID, w)
}

//Get returns the list of apps. You can optionally filter by active apps only.
func Get(w http.ResponseWriter, r *http.Request) {
	activeOnly, _ := strconv.ParseBool(r.FormValue("activeOnly"))

	items, err := db.GetApps(r.Context(), activeOnly)
	if err != nil {
		output.Error(err, "Could not get list of apps.", w)
		return
	}

	output.DataFound(items, w)
}

//Update saves changes to an existing app.
func Update(w http.ResponseWriter, r *http.Request) {
	//Get input data.
	raw := r.FormValue("data")

	//Parse data into struct.
	var a db.App
	err := json.Unmarshal([]byte(raw), &a)
	if err != nil {
		output.Error(err, "Could not parse data to update app.", w)
		return
	}

	//Make sure this isn't being called to add an app.
	if a.ID < 1 {
		output.ErrorAlreadyExists("Could not determine if you are adding or updating an app.", w)
		return
	}

	//Validate.
	errMsg, err := a.Validate(r.Context())
	if err != nil && errMsg != "" {
		output.Error(err, errMsg, w)
		return
	} else if err != nil {
		output.Error(err, "Could not validate data about this app.", w)
		return
	} else if errMsg != "" {
		output.ErrorInputInvalid(errMsg, w)
		return
	}

	//Save.
	err = a.Update(r.Context())
	if err != nil {
		output.Error(err, "Could not update app.", w)
		return
	}

	output.UpdateOK(w)
}
