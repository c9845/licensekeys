/*
Package license handles creation and retrieval of license data. This also handles some
updates to existing licenses.
*/
package license

import (
	"database/sql"
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v4/apikeys"
	"github.com/c9845/licensekeys/v4/config"
	"github.com/c9845/licensekeys/v4/db"
	"github.com/c9845/licensekeys/v4/licensefile"
	"github.com/c9845/licensekeys/v4/timestamps"
	"github.com/c9845/licensekeys/v4/users"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
	"gopkg.in/guregu/null.v3"
)

// This file handles the basic adding of a license, viewing of license data, download
// of the license, and other basic tasks.

// All gets the list of licenses, optionally filtered if needed.
func All(w http.ResponseWriter, r *http.Request) {
	//An App ID of 0 means "look up results for all apps". We translate an invalid
	//ID of < 0 to 0.
	appID, _ := strconv.ParseInt(r.FormValue("appID"), 10, 64)
	if appID < 0 {
		appID = 0
	}

	//Make sure a valid limit value was provided.
	limit, _ := strconv.ParseInt(r.FormValue("limit"), 10, 64)
	if limit < 1 {
		limit = 20
	}

	activeOnly, _ := strconv.ParseBool(r.FormValue("activeOnly"))

	//Look up licenses.
	offset := config.GetTimezoneOffsetForSQLite()
	cols := sqldb.Columns{
		db.TableLicenses + ".ID",
		db.TableLicenses + ".DatetimeCreated",
		db.TableLicenses + ".AppName",
		db.TableLicenses + ".CompanyName",
		db.TableLicenses + ".IssueDate",
		db.TableLicenses + ".ExpirationDate",
		db.TableLicenses + ".Verified",
		db.TableLicenses + ".Active",

		"julianday(" + db.TableLicenses + ".ExpirationDate) < julianday('now') AS Expired",

		//Note the mismatch table and column. This is because we want "what license
		//was this license renewed TO" and "what license was this license renewed
		//"FROM".
		"rrFrom.ToLicenseID AS RenewedToLicenseID",
		"rrTo.FromLicenseID AS RenewedFromLicenseID",

		//Convert dates to timezone in config file which is more applicable to users.
		`datetime(` + db.TableLicenses + `.DatetimeCreated, '` + offset + `') AS DatetimeCreatedInTZ`,
	}
	lics, err := db.GetLicenses(r.Context(), appID, limit, activeOnly, cols)
	if err != nil {
		output.Error(err, "Could not look up list of licenses.", w)
		return
	}

	output.DataFound(lics, w)
}

// One gets the full data for one license.
func One(w http.ResponseWriter, r *http.Request) {
	licenseID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if licenseID < 1 {
		output.ErrorInputInvalid("The license ID provided is invalid.", w)
		return
	}

	offset := config.GetTimezoneOffsetForSQLite()
	cols := sqldb.Columns{
		db.TableLicenses + ".*",
		db.TableUsers + ".Username AS CreatedByUsername",
		db.TableAPIKeys + ".Description AS CreatedByAPIKeyDescription",

		"julianday(" + db.TableLicenses + ".ExpirationDate) < julianday('now') AS Expired",

		db.TableKeypairs + ".KeypairAlgo",
		db.TableKeypairs + ".FingerprintAlgo",
		db.TableKeypairs + ".EncodingAlgo",

		db.TableKeypairs + ".PublicID AS KeypairPublicID",
		db.TableApps + ".PublicID AS AppPublicID",

		//Note the mismatch table and column. This is because we want "what license
		//was this license renewed TO" and "what license was this license renewed
		//"FROM".
		"rrFrom.ToLicenseID AS RenewedToLicenseID",
		"rrTo.FromLicenseID AS RenewedFromLicenseID",

		//Convert dates to timezone in config file which is more applicable to users.
		`datetime(` + db.TableLicenses + `.DatetimeCreated, '` + offset + `') AS DatetimeCreatedInTZ`,
		`DATE(datetime(` + db.TableLicenses + `.DatetimeCreated, '` + offset + `')) AS IssueDateInTZ`,
	}
	l, err := db.GetLicense(r.Context(), licenseID, cols)
	if err == sql.ErrNoRows {
		output.ErrorInputInvalid("The license ID provided does not exist.", w)
		return
	} else if err != nil {
		output.Error(err, "Could not look up license data.", w)
		return
	}

	//Get timezone from config file for displaying in GUI to provide users a bit more
	//context about what DatetimeCreated is.
	l.Timezone = config.Data().Timezone

	output.DataFound(l, w)
}

// Download retrieves the license data as a text file. This file is complete, is is
// signed, and is the license you would distribute for use in your apps.
//
// This can also be used to display the license data in the browser if needed. This is
// really only done for diagnostics by an admin.
func Download(w http.ResponseWriter, r *http.Request) {
	//Get data for license.
	licenseID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if licenseID < 1 {
		output.ErrorInputInvalid("The license ID provided is invalid.", w)
		return
	}

	cols := sqldb.Columns{
		db.TableLicenses + ".*",
		"julianday(" + db.TableLicenses + ".ExpirationDate) < julianday('now') AS Expired",
		db.TableApps + ".Name AS AppName",
		db.TableApps + ".DownloadFilename AS AppDownloadFilename",
	}
	l, err := db.GetLicense(r.Context(), licenseID, cols)
	if err == sql.ErrNoRows {
		output.ErrorInputInvalid("The license ID provided does not exist.", w)
		return
	} else if err != nil {
		output.Error(err, "Could not look up license data.", w)
		return
	} else if l.Expired {
		output.ErrorInputInvalid("This license is expired and cannot be downloaded.", w)
		return
	} else if !l.Active {
		output.ErrorInputInvalid("This license is disabled and cannot be downloaded.", w)
		return
	} else if !l.Verified {
		output.ErrorInputInvalid("This license has not been verified and therefore cannot be downloaded. This is a serious error and should be investigated by an administrator.", w)
		return
	}

	//Get custom fields for license.
	cfr, err := db.GetCustomFieldResults(r.Context(), licenseID)
	if err != nil {
		output.Error(err, "Could not look up custom fields for license.", w)
		return
	}

	//Build the license file.
	f, err := buildLicense(l, cfr)
	if err != nil {
		output.Error(err, "Could not build license.", w)
		return
	}

	//Add signature to license. The signature was already created when license was
	//created so we don't need to recalculate it each time the license is downloaded.
	f.Signature = l.Signature

	//Save download history.
	h := db.DownloadHistory{
		DatetimeCreated:  timestamps.YMDHMS(),
		TimestampCreated: time.Now().UnixNano(),
		LicenseID:        licenseID,
	}

	userID, apiKeyID, err := determineCreatedBy(r)
	if err != nil {
		output.Error(err, "Could not determine who made this request.", w)
		return
	}
	if userID > 0 {
		h.CreatedByUserID = null.IntFrom(userID)
	} else if apiKeyID > 0 {
		h.CreatedByAPIKeyID = null.IntFrom(apiKeyID)
	}

	err = h.Insert(r.Context())
	if err != nil {
		//not exiting on error since this isn't an end of the world event
		log.Println("license.Download", "could not save download history", err)
	}

	//Diagnostic info.
	d, _ := f.ExpiresIn()
	w.Header().Add("X-Days-Until-Expired", strconv.FormatFloat(math.Floor(d.Hours()/24), 'f', 0, 64))

	//If the license file is just being displayed, a rarely used by helpful diagnostic
	//function in the GUI, don't mark the returned data as a file for the browser to
	//download. But, add the correct content type for the browser.
	if r.FormValue("display") == "true" {
		w.Header().Add("Content-Type", "text/"+strings.ToLower(licensefile.FileFormat))
	} else {
		w.Header().Add("Content-Disposition", "attachment; filename=\""+l.AppDownloadFilename+"\"")
	}

	//Write out the license file.
	err = f.Write(w)
	if err != nil {
		output.Error(err, "Could not present license.", w)
		return
	}
}

// Disable marks a license as inactive.
func Disable(w http.ResponseWriter, r *http.Request) {
	//Get inputs and validate.
	licenseID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	note := strings.TrimSpace(r.FormValue("note"))

	if licenseID < 1 {
		output.ErrorInputInvalid("Could not determine which license you want to disable.", w)
		return
	}
	if note == "" {
		output.ErrorInputInvalid("You must provide a note describing why you are disabling this license.", w)
		return
	}

	//Check if this license is already disabled.
	cols := sqldb.Columns{db.TableLicenses + ".Active"}
	l, err := db.GetLicense(r.Context(), licenseID, cols)
	if err != nil {
		output.Error(err, "Could not verify if license is already disabled.", w)
		return
	}
	if !l.Active {
		output.ErrorInputInvalid("This license has already been disabled.", w)
		return
	}

	//Mark the license as inactive and save note.
	c := sqldb.Connection()
	tx, err := c.BeginTxx(r.Context(), nil)
	if err != nil {
		output.Error(err, "Could not mark license as disabled and save note (1).", w)
		return
	}
	defer tx.Rollback()

	err = db.DisableLicense(r.Context(), licenseID, tx)
	if err != nil {
		output.Error(err, "Could not mark license as disabled.", w)
		return
	}

	n := db.LicenseNote{
		LicenseID: licenseID,
		Note:      note + " (License was disabled).",
	}

	//Get info about who or what is disabling this license.
	userID, apiKeyID, err := determineCreatedBy(r)
	if err != nil {
		output.Error(err, "Could not determine who made this request.", w)
		return
	}
	if userID > 0 {
		n.CreatedByUserID = null.IntFrom(userID)
	} else if apiKeyID > 0 {
		n.CreatedByAPIKeyID = null.IntFrom(apiKeyID)
	}

	err = n.Insert(r.Context(), tx)
	if err != nil {
		output.Error(err, "Could not add note about disabled license.", w)
		return
	}

	err = tx.Commit()
	if err != nil {
		output.Error(err, "Could not mark license as disabled and save note (2).", w)
		return
	}

	output.UpdateOK(w)
}

// determineCreatedBy gets the ID of who/what is creating something. A userID or
// apiKeyID will be returned, otherwise an error will be returned.
//
// This func was written as a helper for use when downloading a license file (saving
// to download history).
func determineCreatedBy(r *http.Request) (userID, apiKeyID int64, err error) {
	//Only one of apiKeyID and userID will be provided.
	//  - apiKeyID is set in middleware.ExternalAPI().
	//  - userID is set in middleware.Auth().
	keyID := r.Context().Value(apikeys.APIKeyContextKey)
	uID := r.Context().Value(users.UserIDContextKey)

	if keyID != nil {
		apiKeyID = keyID.(int64)

	} else if uID != nil {
		userID = uID.(int64)

	} else {
		err = errUnknownCreatedByID
	}

	return
}

// errUnknownCreatedByID is returned when we could not determine the user or API key
// that made a request. See determineCreatedBy().
var errUnknownCreatedByID = errors.New("license: unknown creator")
