/*
Package customfields handles interacting with custom fields that you add to your
licenses. These are miscellaneous fields not hard-coded by this app that are stored in
the license file and are defined by a user for an app and values are chosen/provided
when a license is created. These fields allow for storing data such as max user count,
flags for enabling certain features, support information, or really anthing in the
license file. The data stored in these fields is part of the signed data in the
license so they cannot be editted by the end-user of your app for which the license
is for.
*/
package customfields

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/c9845/licensekeys/v4/db"
	"github.com/c9845/licensekeys/v4/users"
	"github.com/c9845/output"
)

// This file specifically deals with creating and managing the fields defined for an
// app.

// Add saves a new custom field.
func Add(w http.ResponseWriter, r *http.Request) {
	//Get input data.
	raw := r.FormValue("data")

	//Parse data into struct.
	var cfd db.CustomFieldDefined
	err := json.Unmarshal([]byte(raw), &cfd)
	if err != nil {
		output.Error(err, "Could not parse data to add custom field.", w)
		return
	}

	//Make sure this isn't being called with an already existing custom field.
	if cfd.ID != 0 {
		output.ErrorAlreadyExists("Could not determine if you are adding or updating a custom field.", w)
		return
	}

	//Validate.
	errMsg, err := cfd.Validate(r.Context())
	if err != nil && errMsg != "" {
		output.Error(err, errMsg, w)
		return
	} else if err != nil {
		output.Error(err, "Could not validate data about this field.", w)
		return
	} else if errMsg != "" {
		output.ErrorInputInvalid(errMsg, w)
		return
	}

	//Get user who is adding this field.
	loggedInUserID, err := users.GetUserIDFromRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	cfd.CreatedByUserID = loggedInUserID

	//Save.
	err = cfd.Insert(r.Context())
	if err != nil {
		output.Error(err, "Could not save field.", w)
		return
	}

	output.InsertOK(cfd.ID, w)
}

// GetDefined returns the list of fields for an app. You can optionally filter by active fields only.
func GetDefined(w http.ResponseWriter, r *http.Request) {
	appID, _ := strconv.ParseInt(r.FormValue("appID"), 10, 64)
	activeOnly, _ := strconv.ParseBool(r.FormValue("activeOnly"))

	if appID < 1 {
		output.ErrorInputInvalid("Cannot determine which app you want to get custom fields for.", w)
		return
	}

	items, err := db.GetCustomFieldsDefined(r.Context(), appID, activeOnly)
	if err != nil {
		output.Error(err, "Could not get list of fields.", w)
		return
	}

	output.DataFound(items, w)
}

// Update saves changes to an existing field.
func Update(w http.ResponseWriter, r *http.Request) {
	//Get input data.
	raw := r.FormValue("data")

	//Parse data into struct.
	var cfd db.CustomFieldDefined
	err := json.Unmarshal([]byte(raw), &cfd)
	if err != nil {
		output.Error(err, "Could not parse data to update custom field.", w)
		return
	}

	//Make sure this isn't being called with to add a field.
	if cfd.ID < 1 {
		output.ErrorAlreadyExists("Could not determine if you are adding or updating a custom field.", w)
		return
	}

	//Validate.
	errMsg, err := cfd.Validate(r.Context())
	if err != nil && errMsg != "" {
		output.Error(err, errMsg, w)
		return
	} else if err != nil {
		output.Error(err, "Could not validate data about this field.", w)
		return
	} else if errMsg != "" {
		output.ErrorInputInvalid(errMsg, w)
		return
	}

	//Save.
	err = cfd.Update(r.Context())
	if err != nil {
		output.Error(err, "Could not update field.", w)
		return
	}

	output.UpdateOK(w)
}

// DeleteDefined marks a custom field as inactive. The field will no longer be available
// when creating a license. The field will still be displayed when viewing already
// created licenses.
func DeleteDefined(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)

	if id < 1 {
		output.ErrorInputInvalid("Could not determine which custom field you want to delete.", w)
		return
	}

	cfd := db.CustomFieldDefined{
		ID: id,
	}
	err := cfd.Delete(r.Context())
	if err != nil {
		output.Error(err, "Could not delete custom field.", w)
		return
	}

	output.UpdateOK(w)
}
