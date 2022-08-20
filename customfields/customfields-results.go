/*
Package customfields handles interacting with custom fields that you add to your
licenses. These are miscellaneous fields not hard-coded by this app that are stored in
the license file and are defined by a user for an app and values are chosen/provided
when a license is created. These fields allow for storing data such as max user count,
flags for enabling certain features, support information, or really anthing in the
license file. The data stored in these fields is part of the signed data in the
license so they cannot be editted by the end-user of your app for which the license
is for.

This file specifically deals looking up the values set for a license.
*/
package customfields

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v2"
)

// GetResults looks up the custom field values saved for a license.
func GetResults(w http.ResponseWriter, r *http.Request) {
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

	//Look up the custom field results.
	rr, err := db.GetCustomFieldResults(r.Context(), licenseID)
	if err != nil {
		output.Error(err, "Could not look up custom field results for license.", w)
		return
	}

	output.DataFound(rr, w)
}
