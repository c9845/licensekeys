/*
Package license handles creation and retrieval of license data. This also handles some
updates to existing licenses.

This file specifically deals with the notes for a license. Notes are useful for random
documentation purposes.
*/
package license

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/c9845/licensekeys/db"
	"github.com/c9845/licensekeys/users"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v2"
	"gopkg.in/guregu/null.v3"
)

//Notes gets the list of notes for a license.
func Notes(w http.ResponseWriter, r *http.Request) {
	//Make sure a license ID was provided and it is valid.
	licenseID, _ := strconv.ParseInt(r.FormValue("licenseID"), 10, 64)
	if licenseID < 1 {
		output.ErrorInputInvalid("The license ID provided is invalid.", w)
		return
	}

	cols := sqldb.Columns{db.TableLicenses + ".Active"}
	_, err := db.GetLicense(r.Context(), licenseID, cols)
	if err == sql.ErrNoRows {
		output.ErrorInputInvalid("The license ID provided does not exist.", w)
		return
	} else if err != nil {
		output.Error(err, "Could not look up license data.", w)
		return
	}

	//Look up the notes.
	orderBy := "ORDER BY " + db.TableLicenseNotes + ".DatetimeCreated DESC"
	hh, err := db.GetNotes(r.Context(), licenseID, orderBy)
	if err != nil {
		output.Error(err, "Could not look up license notes.", w)
		return
	}

	output.DataFound(hh, w)
}

//AddNote adds a note for a license.
func AddNote(w http.ResponseWriter, r *http.Request) {
	//Get input data.
	raw := r.FormValue("data")

	//Parse data into struct.
	var n db.LicenseNote
	err := json.Unmarshal([]byte(raw), &n)
	if err != nil {
		output.Error(err, "Could not parse data to add note.", w)
		return
	}

	//Sanitize.
	n.Note = strings.TrimSpace(n.Note)

	//Validate.
	if n.LicenseID < 1 {
		output.ErrorInputInvalid("Could not determine which license you want to add a note for.", w)
		return
	}
	if n.Note == "" {
		output.ErrorInputInvalid("You must provide a note.", w)
		return
	}

	//Get user who is adding this note.
	loggedInUserID, err := users.GetUserIDByRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	n.CreatedByUserID = null.IntFrom(loggedInUserID)

	//Save.
	err = n.Insert(r.Context(), nil)
	if err != nil {
		output.Error(err, "Could not save note.", w)
		return
	}

	output.InsertOK(n.ID, w)
}
