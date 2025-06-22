package license

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v4/apikeys"
	"github.com/c9845/licensekeys/v4/db"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
	"gopkg.in/guregu/null.v3"
)

// Renew creates a new license from an existing license, by copying the existing
// license's data and assigning a new expiration date.
//
// Renewal can be performed via the GUI or API.
//
// The original license is disabled so it cannot be mistakenly downloaded.
func Renew(w http.ResponseWriter, r *http.Request) {
	//Handle license ID.
	//
	//If this request is from the app GUI, the licenseID will be the lower-level,
	//integer ID.
	//
	//If this request is from the public API, the licenseID will be the public UUID.
	var fromLicenseID int64
	if strings.Contains(r.URL.Path, "api/v1/") && r.Context().Value(apikeys.APIKeyContextKey) != nil {
		//Public API!
		fromLicensePublicID := strings.TrimSpace(r.FormValue("id"))
		if fromLicensePublicID == "" {
			output.ErrorInputInvalid("Could not determine which license you want to renew.", w)
			return
		}

		//Look up license's private ID.
		cols := sqldb.Columns{db.TableLicenses + ".ID"}
		l, err := db.GetLicenseByPublicID(r.Context(), db.UUID(fromLicensePublicID), cols)
		if err != nil {
			output.Error(err, "Could not look up license.", w)
			return
		}
		fromLicenseID = l.ID

	} else {
		//Private request from app's GUI.
		fromLicenseID, _ = strconv.ParseInt(r.FormValue("id"), 10, 64)
	}

	if fromLicenseID < 1 {
		output.ErrorInputInvalid("Could not determine which license you want to renew.", w)
		return
	}

	//Get new expiration date.
	newExpirationDateStr := strings.TrimSpace(r.FormValue("newExpirationDate"))
	if newExpirationDateStr == "" {
		output.ErrorInputInvalid("You must provide the new expiration date.", w)
		return
	}
	newExpirationDate, err := time.Parse("2006-01-02", newExpirationDateStr)
	if err != nil {
		output.Error(err, "You must provide a new expiration date in YYYY-MM-DD format.", w)
		return
	}

	//Look up the existing license.
	//
	//Confirm it hasn't been disabled and that the new expiration date is after the
	//current expiration date.
	//
	//We also need this data for copying to a new license.
	cols := sqldb.Columns{
		db.TableLicenses + ".*",
	}
	fromLicense, err := db.GetLicense(r.Context(), fromLicenseID, cols)
	if err != nil {
		output.Error(err, "Could not look up existing license's data.", w)
		return
	}
	if !fromLicense.Active {
		output.ErrorInputInvalid("This license has been disabled and cannot be renewed.", w)
		return
	}
	existingExpirationDate, err := time.Parse("2006-01-02", fromLicense.ExpirationDate)
	if err != nil {
		output.Error(err, "Could not confirm if new expiration date is after existing license's expiration date.", w)
		return
	}
	if newExpirationDate.Before(existingExpirationDate) {
		output.ErrorInputInvalid("The new expiration date must be after the existing license's expiration date, "+fromLicense.ExpirationDate+".", w)
		return
	}

	//Make sure this license hasn't already been renewed. A license can only be
	//renewed once.
	_, err = db.GetRenewalRelationshipByFromID(r.Context(), fromLicenseID)
	if err != nil && err != sql.ErrNoRows {
		output.Error(err, "Could not determine if this license has already been renewed.", w)
		return
	} else if err == nil {
		output.ErrorInputInvalid("This license has already been renewed.", w)
		return
	}

	//Make sure the app for this license is still active.
	app, err := db.GetAppByID(r.Context(), fromLicense.AppID)
	if err != nil {
		output.Error(err, "Could not look up app.", w)
		return
	}
	if !app.Active {
		output.ErrorInputInvalid("This app is not active, you cannot renew licenses for it.", w)
		return
	}

	//Make sure the keypair used for the existing license is still active. The
	//keypair could have been deleted after this license was created but before
	//this renewal and we don't want to use a deleted keypair since it may have been
	//hacked or something.
	kp, err := db.GetKeypairByID(r.Context(), fromLicense.KeyPairID)
	if err != nil {
		output.Error(err, "Could not look up keypair.", w)
		return
	}
	if !kp.Active {
		output.ErrorInputInvalid("The keypair used to sign this license has been disabled, you cannot renew this license. Create a new license and choose a new keypair.", w)
		return
	}

	//Copy the "from" license's data to use for the "to" license.
	toLicense := fromLicense

	//Unset created-by fields from "from" license.
	toLicense.CreatedByAPIKeyID = null.IntFrom(0)
	toLicense.CreatedByUserID = null.IntFrom(0)

	//Unset some other fields from "from" license. Just to prevent mistakes.
	toLicense.ID = 0
	toLicense.DatetimeCreated = ""
	toLicense.IssueDate = ""
	toLicense.IssueTimestamp = 0
	toLicense.Fingerprint = ""
	toLicense.Signature = ""

	//Save the new expiration date.
	toLicense.ExpirationDate = newExpirationDateStr

	//Get info about who or what is renewing this license.
	userID, apiKeyID, err := determineCreatedBy(r)
	if err != nil {
		output.Error(err, "Could not determine who made this request.", w)
		return
	}
	if userID > 0 {
		toLicense.CreatedByUserID = null.IntFrom(userID)
	} else if apiKeyID > 0 {
		toLicense.CreatedByAPIKeyID = null.IntFrom(apiKeyID)
	}

	//Look up custom fields for "from" license.
	cfr, err := db.GetCustomFieldResults(r.Context(), fromLicenseID)
	if err != nil {
		output.Error(err, "Could not look up custom fields.", w)
		return
	}

	//Unset data in custom fields to prevent mistakes.
	for idx, field := range cfr {
		field.ID = 0
		field.LicenseID = 0
		field.DatetimeCreated = ""
		cfr[idx] = field
	}

	//Create and save new, renewed, license.
	toLicenseID, renewedFile, err := create(r.Context(), toLicense, cfr, app, kp, fromLicenseID)
	if err != nil {
		output.Error(err, "Could not renew license.", w)
		return
	}

	//Check if user wants the actual license file returned. This is typically done so
	//that a second request to get the license file isn't needed.
	if r.FormValue("returnLicenseFile") == "true" {
		//Set filename for downloading.
		w.Header().Add("Content-Disposition", "inline; filename=\""+app.DownloadFilename+"\"")

		//Return the file.
		err = renewedFile.Write(w)
		if err != nil {
			output.Error(err, "Could not return license file.", w)
			return
		}

		//Save download history.
		h := db.DownloadHistory{
			TimestampCreated:  time.Now().UnixNano(),
			LicenseID:         toLicenseID,
			CreatedByUserID:   null.IntFrom(toLicense.CreatedByUserID.Int64),
			CreatedByAPIKeyID: null.IntFrom(toLicense.CreatedByAPIKeyID.Int64),
		}

		err = h.Insert(r.Context())
		if err != nil {
			//not exiting on error since this isn't an end of the world event
			log.Println("license.CreateViaPublicAPI", "could not save download history", err)
		}

		return

	}

	output.InsertOK(toLicenseID, w)
}
