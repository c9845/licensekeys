/*
Package license handles creation and retrieval of license data. This also handles some
updates to existing licenses.
*/
package license

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/keypairs"
	"github.com/c9845/licensekeys/v2/licensefile"
	"github.com/c9845/licensekeys/v2/middleware"
	"github.com/c9845/licensekeys/v2/timestamps"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
	"gopkg.in/guregu/null.v3"
)

// This file handles the basic adding of a license, viewing of license data, download
// of the license, and other basic tasks.

// AddViaAPI handles transforming the data provided by the request to create a license
// via the public API to the format the interal API/Add func expects. This is done to
// keep the public API simpler: the user building the request can provide the main
// license data in key:value pairs rather than a struct/object, and the user can
// provide the list of custom fields in an object of field-name:value pairings rather
// than an array of objects. This takes the user provided data and builds the internal
// license struct and slice of custom field results structs before calling Add() to
// handle the actual validation and saving of the license.
//
// Note that when adding a license via the public API, the request can have either the
// AppID or KeyPairID. If the AppID is provided, then the default key pair's ID is used.
// If the KeyPairID is provided, we simply use the parent app's ID.
func AddViaAPI(w http.ResponseWriter, r *http.Request) {
	//Read input data and build license object.
	appID, _ := strconv.ParseInt(r.FormValue("appID"), 10, 64)
	keyPairID, _ := strconv.ParseInt(r.FormValue("keyPairID"), 10, 64)

	l := db.License{
		AppID:       appID,
		KeyPairID:   keyPairID,
		CompanyName: r.FormValue("companyName"),
		ContactName: r.FormValue("contactName"),
		PhoneNumber: r.FormValue("phoneNumber"),
		Email:       r.FormValue("email"),
		ExpireDate:  r.FormValue("expireDate"),
	}
	encoded, err := json.Marshal(l)
	if err != nil {
		output.Error(err, "Could not build license data.", w)
		return
	}
	r.Form.Add("licenseData", string(encoded))

	//If user provided key pair's ID, then look up parent app's ID. We need the app ID
	//to look up the list of defined custom fields for building the slice of custom
	//field results (translating the provided object of field's name-value pairs or
	//using default field values).
	//
	//If user provided an app's ID, then we don't have to do anything. When the license
	//data is validated in Add(), the default key pair's ID will be retrieved based on
	//the provided app ID.
	if keyPairID < 1 && l.AppID < 1 {
		output.ErrorInputInvalid("Missing appID and keyPairID. One of these must be provided.", w)
		return
	}
	if l.KeyPairID > 0 {
		kp, err := db.GetKeyPairByID(r.Context(), l.KeyPairID)
		if err != nil {
			output.Error(err, "Could not look up app for keyPairID.", w)
			return
		}
		l.AppID = kp.AppID
	}

	//Look up custom fields defined for app. We need these to get field name to match
	//up against the fields provided and to get type for each field to build slice
	//correctly.
	definedFields, err := db.GetCustomFieldsDefined(r.Context(), l.AppID, true)
	if err != nil {
		output.Error(err, "Could not look up custom fields for translation.", w)
		return
	}

	//Custom fields are provided as a set of key-value pairs in an object. But the Add
	//func expects an array of objects, one for each field, with appropriate fields
	//set. This will handle the translation to the format expected by Add(). The
	//QueryUnescape is needed since the value for this field is a url encoded JSON
	//object, the url encoding is needed to pass the { } characters successfully.
	rawObject, err := url.QueryUnescape(r.FormValue("fields"))
	if err != nil {
		output.Error(err, "Could not parse custom fields.", w)
		return
	}

	var object map[string]interface{}
	err = json.Unmarshal([]byte(rawObject), &object)
	if err != nil && len(rawObject) > 0 {
		//We check the length to handle times when the request didn't provide any
		//custom field data. In this case, we can ignore the json unmarshal error
		//since we will just use the default value for every custom field.
		output.Error(err, "Could not parse custom fields for translation.", w)
		return
	}

	var cfr []db.CustomFieldResult
	for _, definedField := range definedFields {
		//Build the base object for the field to be saved.
		c := db.CustomFieldResult{
			CustomFieldName:      definedField.Name,
			CustomFieldDefinedID: definedField.ID,
		}

		//Find the matching value provided via the API request.
		matchFound := false
		for key, value := range object {
			matchFound = true

			if definedField.Name == key {
				switch definedField.Type {
				case db.CustomFieldTypeInteger:
					c.IntegerValue = null.IntFrom(int64(value.(float64)))
				case db.CustomFieldTypeDecimal:
					c.DecimalValue = null.FloatFrom(value.(float64))
				case db.CustomFieldTypeText:
					c.TextValue = null.StringFrom(value.(string))
				case db.CustomFieldTypeBoolean:
					c.BoolValue = null.BoolFrom(value.(bool))
				case db.CustomFieldTypeMultiChoice:
					c.MultiChoiceValue = null.StringFrom(value.(string))
				case db.CustomFieldTypeDate:
					c.DateValue = null.StringFrom(value.(string))
				default:
					//This will never be hit because we looked up defined fields from
					//db and these should always have valid types (unless db was
					//modified manually).
				}
			}
		} //end for: find matching value for field name.

		//Handle if no matching field was provided. In this case, we will just use the
		//default value set for the field.
		if !matchFound {
			switch definedField.Type {
			case db.CustomFieldTypeInteger:
				c.IntegerValue = null.IntFrom(definedField.IntegerDefaultValue.Int64)
			case db.CustomFieldTypeDecimal:
				c.DecimalValue = null.FloatFrom(definedField.DecimalDefaultValue.Float64)
			case db.CustomFieldTypeText:
				c.TextValue = null.StringFrom(definedField.TextDefaultValue.String)
			case db.CustomFieldTypeBoolean:
				c.BoolValue = null.BoolFrom(definedField.BoolDefaultValue.Bool)
			case db.CustomFieldTypeMultiChoice:
				c.MultiChoiceValue = null.StringFrom(definedField.MultiChoiceDefaultValue.String)
			case db.CustomFieldTypeDate:
				now := time.Now()
				add := now.AddDate(0, 0, int(definedField.DateDefaultIncrement.Int64))
				c.DateValue = null.StringFrom(add.Format("2006-01-02"))
			default:
				//This will never be hit because we looked up defined fields from db
				//and these should always have valid types (unless db was modified
				//manually).
			}

			// log.Println("license.AddViaAPI", "using default", definedField.Name)
		} //end if: use default value for not provided field

		cfr = append(cfr, c)
	} //end for: loop through defined fields

	//Encode the slice of custom field stucts to json and store them in the request
	//so that we can simply call the Add() func now.
	encoded, err = json.Marshal(cfr)
	if err != nil {
		output.Error(err, "Could not build translated fields.", w)
		return
	}
	r.Form.Add("customFields", string(encoded))

	//Call Add to add a license like is done via the GUI. The Add func will handle any
	//API specific stuff (setting CreatedByAPIKeyID vs CreatedByUserID) as well as do
	//much more validation.
	Add(w, r)
}

// Add saves the data used to create a license. First, we handle saving the common
// license data, then we handle saving the custom field results, then we generate the
// license file and sign it, we update the saved license data with the signature, and
// finally we verify the license file by creating it, rereading it, and checking the
// signature with the public key.
func Add(w http.ResponseWriter, r *http.Request) {
	//Parse and validate main license data.
	rawCommonData := r.FormValue("licenseData")
	var l db.License
	err := json.Unmarshal([]byte(rawCommonData), &l)
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

	//Get key pair data. We need this to get the app ID of the app to look up since
	//we need to save some details about the app for this license. We also need this
	//to get the private key info to sign the license file since we create the
	//signature now, not when a license is downloaded, for efficiency purposes.
	kp, err := db.GetKeyPairByID(r.Context(), l.KeyPairID)
	if err != nil {
		output.Error(err, "Could not look up signature details.", w)
		return
	}
	if !kp.Active {
		output.ErrorInputInvalid("This key pair is not active. Please choose an active key pair for signing this license.", w)
		return
	}

	//Get app data. We need this for the file format, signature hash algorithm and the
	//encoding type.
	a, err := db.GetAppByID(r.Context(), kp.AppID)
	if err != nil {
		output.Error(err, "Could not look up app details this license is for.", w)
		return
	}
	if !a.Active {
		output.ErrorInputInvalid("This app is not active. You cannot create a license for it.", w)
		return
	}

	//Parse and validate custom fields. This isn't done immediately after validating
	//main license data because we need the app's ID.
	rawCustomFields := r.FormValue("customFields")
	var fields db.MultiCustomFieldResult
	err = json.Unmarshal([]byte(rawCustomFields), &fields)
	if err != nil {
		output.Error(err, "Could not parse custom fields to create license.", w)
		return
	}

	errMsg, err = fields.Validate(r.Context(), a.ID, false)
	if err != nil && errMsg != "" {
		output.Error(err, errMsg, w)
		return
	} else if err != nil {
		output.Error(err, "Could not validate field data about this license.", w)
		return
	} else if errMsg != "" {
		output.ErrorInputInvalid(errMsg, w)
		return
	}

	//Get info about who or what is creating this license.
	userID, apiKeyID, err := getCreatedBy(r)
	if err != nil {
		output.Error(err, "Could not determine who made this request.", w)
		return
	}
	if userID > 0 {
		l.CreatedByUserID = null.IntFrom(userID)
	} else if apiKeyID > 0 {
		l.CreatedByAPIKeyID = null.IntFrom(apiKeyID)
	}

	//Set the license creation timestamps and some other data. We save some app
	//related data in case the app's details are changed in the future since once a
	//license is created, and the signature is created, we need the same details to
	//download a valid license any time in the future.
	l.IssueDate = timestamps.YMD()
	l.IssueTimestamp = time.Now().Unix()
	l.AppName = a.Name
	l.FileFormat = a.FileFormat
	l.ShowLicenseID = a.ShowLicenseID
	l.ShowAppName = a.ShowAppName

	//Get DatetimeCreated value. This way we will have the exact same value for the
	//license, custom field results, etc.
	datetimeCreated := timestamps.YMDHMS()
	l.DatetimeCreated = datetimeCreated

	//Start transaction since we are saving multiple things.
	c := sqldb.Connection()
	tx, err := c.BeginTxx(r.Context(), nil)
	if err != nil {
		output.Error(err, "Could not save license data (1).", w)
		return
	}
	defer tx.Rollback()

	//Save main license data. This will get us the license ID which we need to save
	//the custom field results and possible for use in the license if required per
	//the app's details.
	err = l.Insert(r.Context(), tx)
	if err != nil {
		output.Error(err, "Could not save license data (2).", w)
		return
	}

	//Save custom field results.
	for _, field := range fields {
		if userID > 0 {
			field.CreatedByUserID = null.IntFrom(userID)
		} else if apiKeyID > 0 {
			field.CreatedByAPIKeyID = null.IntFrom(apiKeyID)
		}

		field.LicenseID = l.ID
		field.DatetimeCreated = datetimeCreated

		innerErr := field.Insert(r.Context(), tx)
		if innerErr != nil {
			output.Error(innerErr, "Could not save field \""+field.CustomFieldName+"\" therefore license could not be saved.", w)
			return
		}
	}

	//Create the new license file.
	f, err := buildLicense(l, fields)
	if err != nil {
		output.Error(err, "Could not build license for signing and verification.", w)
		return
	}

	//Decrypt the private key, if needed.
	privateKey := []byte(kp.PrivateKey)
	if kp.PrivateKeyEncrypted {
		encKey := config.Data().PrivateKeyEncryptionKey

		pk, err := hex.DecodeString(kp.PrivateKey)
		if err != nil {
			output.Error(err, "Could not decrypt private key to sign license data (1).", w)
			return
		}

		decryptedPrivKey, err := keypairs.DecryptPrivateKey(encKey, pk)
		if err != nil {
			output.Error(err, "Could not decrypt private key to sign license data (2).", w)
			return
		}
		privateKey = decryptedPrivKey
	}

	//Sign the license file.
	err = f.Sign(privateKey, kp.AlgorithmType)
	if err != nil {
		output.Error(err, "Could not generate signature.", w)
		return
	}

	//Save the signature
	l.Signature = f.Signature
	err = l.SaveSignature(r.Context(), tx)
	if err != nil {
		output.Error(err, "Could not save signature.", w)
		return
	}

	//Commit now to save the license even though we don't know if it is can be
	//successfully validated with the public key.
	err = tx.Commit()
	if err != nil {
		output.Error(err, "Could not complete saving of new license.", w)
		return
	}

	//Verify the just created license data and signature. This "writes" out the
	//complete license file with signature and then "reads" it like a third-party app
	//would to verify the signature with a public key. This is done to confirm the
	//signature is valid.
	err = writeReadVerify(f, kp.AlgorithmType, []byte(kp.PublicKey))
	if err == licensefile.ErrBadSignature {
		output.Error(licensefile.ErrBadSignature, "License could not be verified and therefore cannot be used. Please contact an administrator and have them investigate this error.", w)
		return
	} else if err != nil {
		output.Error(err, "An error occured while trying to verify the license. Please ask an administrator to investigate this error.", w)
		return
	}

	//Mark the license as verified.
	l.Verified = true
	err = l.MarkVerified(r.Context())
	if err != nil {
		output.Error(err, "Could not mark license as valid.", w)
		return
	}

	//Check if user wants the actual license returned. This is typically only for
	//public API requests and is done so that a second request to get the license file
	//isn't needed.
	if r.FormValue("returnLicenseFile") == "true" {
		//Set suggested filename.
		filename := replaceFilenamePlaceholders(a.DownloadFilename, l.ID, a.Name, a.FileFormat)
		w.Header().Add("Content-Disposition", "inline; filename=\""+filename+"\"")

		err = f.Write(w)
		if err != nil {
			output.Error(err, "Could not return license file.", w)
			return
		}

		//Save download history.
		h := db.DownloadHistory{
			DatetimeCreated:   datetimeCreated,
			TimestampCreated:  time.Now().UnixNano(),
			LicenseID:         l.ID,
			CreatedByUserID:   null.IntFrom(l.CreatedByUserID.Int64),
			CreatedByAPIKeyID: null.IntFrom(l.CreatedByAPIKeyID.Int64),
		}

		err = h.Insert(r.Context())
		if err != nil {
			//not exiting on error since this isn't an end of the world event
			log.Println("license.Add", "could not save download history", err)
		}

		return
	}

	//Return just the new license's ID. The GUI will use this to redirect the user to
	//the license's management page if license key was created in the GUI.
	output.InsertOK(l.ID, w)
}

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
		db.TableLicenses + ".ExpireDate",
		db.TableLicenses + ".Verified",
		db.TableLicenses + ".Active",

		"julianday(" + db.TableLicenses + ".ExpireDate) < julianday('now') AS Expired",

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
		db.TableKeyPairs + ".AlgorithmType AS KeyPairAlgoType",

		"julianday(" + db.TableLicenses + ".ExpireDate) < julianday('now') AS Expired",

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
		"julianday(" + db.TableLicenses + ".ExpireDate) < julianday('now') AS Expired",
		db.TableApps + ".Name AS AppName",
		db.TableApps + ".DownloadFilename AS AppDownloadFilename",
		db.TableApps + ".FileFormat AS AppFileFormat",
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

	userID, apiKeyID, err := getCreatedBy(r)
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

	//Replace any placeholders in the download filename. Placeholders are special
	//words wrapped in {} characters provided for the license's app.
	filename := replaceFilenamePlaceholders(l.AppDownloadFilename, l.ID, l.AppName, l.AppFileFormat)

	//Diagnostic info.
	d, _ := f.ExpiresIn()
	w.Header().Add("X-Days-Until-Expired", strconv.FormatFloat(math.Floor(d.Hours()/24), 'f', 0, 64))
	w.Header().Add("Content-Disposition", "attachment; filename=\""+filename+"\"")

	//Write out the license file.
	err = f.Write(w)
	if err != nil {
		output.Error(err, "Could not present license.", w)
		return
	}
}

// buildLicense builds the File with the required data. The resulting File would need
// to be signed, or have an already calculated signature added, and marshalled to the
// required file format.
//
// This is used when a license is created so it can be validated and when downloading
// a license.
func buildLicense(l db.License, cfr []db.CustomFieldResult) (f licensefile.File, err error) {
	//Make sure required fields for building license file were provided.
	err = l.FileFormat.Valid()
	if err != nil {
		return
	}

	//Set common data in file.
	f = licensefile.File{
		CompanyName:    l.CompanyName,
		ContactName:    l.ContactName,
		PhoneNumber:    l.PhoneNumber,
		Email:          l.Email,
		IssueDate:      l.IssueDate,
		IssueTimestamp: l.IssueTimestamp,
		ExpireDate:     l.ExpireDate,
	}

	//these fields are just used for the signing process
	f.SetFileFormat(l.FileFormat)

	//Set optional fields.
	if l.ShowLicenseID {
		f.LicenseID = l.ID
	}
	if l.ShowAppName {
		f.AppName = l.AppName
	}

	//Add the custom field results as a map to the file.
	metadata := make(map[string]any, len(cfr))
	for _, f := range cfr {
		switch f.CustomFieldType {
		case db.CustomFieldTypeInteger:
			metadata[f.CustomFieldName] = f.IntegerValue.Int64
		case db.CustomFieldTypeDecimal:
			metadata[f.CustomFieldName] = f.DecimalValue.Float64
		case db.CustomFieldTypeText:
			metadata[f.CustomFieldName] = f.TextValue.String
		case db.CustomFieldTypeBoolean:
			metadata[f.CustomFieldName] = f.BoolValue.Bool
		case db.CustomFieldTypeMultiChoice:
			metadata[f.CustomFieldName] = f.MultiChoiceValue.String
		case db.CustomFieldTypeDate:
			metadata[f.CustomFieldName] = f.DateValue.String
		default:
			//This should never be hit since we validated field types when they
			//were saved/defined.
		}
	}
	f.Metadata = metadata

	return
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
	userID, apiKeyID, err := getCreatedBy(r)
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

// Renew creates a new license from an existing license, just with a new expiration
// date set. This creates a copy of the existing license's common data and custom
// field results. The renewal relationship is also saved to link the licenses together.
//
// The original license is disabled so it cannot be mistakenly downloaded.
func Renew(w http.ResponseWriter, r *http.Request) {
	//Get inputs and validate.
	fromLicenseID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	newExpireDateStr := strings.TrimSpace(r.FormValue("newExpireDate"))

	//Validate.
	if fromLicenseID < 1 {
		output.ErrorInputInvalid("Could not determine which license you want to renew.", w)
		return
	}
	if newExpireDateStr == "" {
		output.ErrorInputInvalid("You must provide the new expiration date.", w)
		return
	}
	newExpireDate, err := time.Parse("2006-01-02", newExpireDateStr)
	if err != nil {
		output.Error(err, "You must provide a new expiration date in YYYY-MM-DD format.", w)
		return
	}

	//Look up existing license data so we can confirm it hasn't been disabled and
	//that the new expiration date is after the current expiration date.
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
	existingExpireDate, err := time.Parse("2006-01-02", fromLicense.ExpireDate)
	if err != nil {
		output.Error(err, "Could not confirm if new expiration date is after existing license's expiration date.", w)
		return
	}
	if newExpireDate.Before(existingExpireDate) {
		output.ErrorInputInvalid("The new expiration date must be after the existing license's expiration date, "+fromLicense.ExpireDate+".", w)
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

	//Get a copy of the "from" license's data to use for the "to" license.
	toLicense := fromLicense

	//Unset created-by fields from old/copy-from license.
	toLicense.CreatedByAPIKeyID = null.IntFrom(0)
	toLicense.CreatedByUserID = null.IntFrom(0)

	//Get info about who or what is renewing this license.
	userID, apiKeyID, err := getCreatedBy(r)
	if err != nil {
		output.Error(err, "Could not determine who made this request.", w)
		return
	}
	if userID > 0 {
		toLicense.CreatedByUserID = null.IntFrom(userID)
	} else if apiKeyID > 0 {
		toLicense.CreatedByAPIKeyID = null.IntFrom(apiKeyID)
	}

	//Get DatetimeCreated value. This way we will have the exact same value for the
	//license, custom field results, and renewal relationship.
	datetimeCreated := timestamps.YMDHMS()

	//Modify existing license data for new license.
	toLicense.ID = 0                             //this will be populated with the new license's ID in Insert().
	toLicense.ExpireDate = newExpireDateStr      //new user provided date.
	toLicense.DatetimeModified = ""              //license hasn't been modified, so unset this to reduce confusion.
	toLicense.IssueDate = timestamps.YMD()       //
	toLicense.IssueTimestamp = time.Now().Unix() //
	toLicense.Signature = ""                     //will be set later...
	toLicense.DatetimeCreated = datetimeCreated

	//Start transaction since we are saving multiple things.
	c := sqldb.Connection()
	tx, err := c.BeginTxx(r.Context(), nil)
	if err != nil {
		output.Error(err, "Could not save renewed license (1).", w)
		return
	}
	defer tx.Rollback()

	//Save common data. This will get us the license ID which we need to save the
	//custom field results and possible for use in the license if required per the
	//app's details.
	err = toLicense.Insert(r.Context(), tx)
	if err != nil {
		output.Error(err, "Could not save renewed license (2).", w)
		return
	}

	//Copy each custom field result.
	ff, err := db.GetCustomFieldResults(r.Context(), fromLicenseID)
	if err != nil {
		output.Error(err, "Could not look up existing license's custom field results.", w)
		return
	}

	for _, f := range ff {
		if userID > 0 {
			f.CreatedByUserID = null.IntFrom(userID)
			f.CreatedByAPIKeyID = null.IntFrom(0) //unset from old/copy-from license just in case.
		} else if apiKeyID > 0 {
			f.CreatedByAPIKeyID = null.IntFrom(apiKeyID)
			f.CreatedByUserID = null.IntFrom(0) //unset from old/copy-from license just in case.
		}

		f.LicenseID = toLicense.ID
		f.DatetimeCreated = datetimeCreated

		innerErr := f.Insert(r.Context(), tx)
		if innerErr != nil {
			output.Error(innerErr, "Could not save field \""+f.CustomFieldName+"\" therefore license could not be saved.", w)
			return
		}
	}

	//Create the renewal relationship.
	relationship := db.RenewalRelationship{
		FromLicenseID:   fromLicenseID,
		ToLicenseID:     toLicense.ID,
		DatetimeCreated: datetimeCreated,
	}

	if userID > 0 {
		relationship.CreatedByUserID = null.IntFrom(userID)
	} else if apiKeyID > 0 {
		relationship.CreatedByAPIKeyID = null.IntFrom(apiKeyID)
	}

	err = relationship.Insert(r.Context(), tx)
	if err != nil {
		output.Error(err, "Could not save renewal relationship.", w)
		return
	}

	//Disable the "renewed-from" license so that it cannot be mistakenly downloaded.
	err = db.DisableLicense(r.Context(), fromLicenseID, tx)
	if err != nil {
		//Don't return on this error since it isn't the end of the world.
		log.Println("license.Renew", "could not mark 'from' license as disabled, skipping", err)
	}

	//
	//All the db queries needed to copy data have occured, the renewal license now
	//exists. However, we still need to perform the other "after inserting a license"
	//stuff.
	//

	//Get key pair data. We need this to get the private key info to sign the license
	//files since we create the signature now, not when a license is downloaded, for
	//efficiency purposes.
	kp, err := db.GetKeyPairByID(r.Context(), toLicense.KeyPairID)
	if err != nil {
		output.Error(err, "Could not look up signature details.", w)
		return
	}
	if !kp.Active {
		output.ErrorInputInvalid("This key pair used for the original license is no longer active. This license cannot be renewed.", w)
		return
	}

	//Create the renewal license file.
	f, err := buildLicense(toLicense, ff)
	if err != nil {
		output.Error(err, "Could not build license for signing and verification.", w)
		return
	}

	//Decrypt the private key, if needed.
	privateKey := []byte(kp.PrivateKey)
	if kp.PrivateKeyEncrypted {
		encKey := config.Data().PrivateKeyEncryptionKey

		pk, err := hex.DecodeString(kp.PrivateKey)
		if err != nil {
			output.Error(err, "Could not decrypt private key to sign license data (1).", w)
			return
		}

		decryptedPrivKey, err := keypairs.DecryptPrivateKey(encKey, pk)
		if err != nil {
			output.Error(err, "Could not decrypt private key to sign license data (2).", w)
			return
		}
		privateKey = decryptedPrivKey
	}

	//Sign the license file.
	err = f.Sign(privateKey, kp.AlgorithmType)
	if err != nil {
		output.Error(err, "Could not generate signature.", w)
		return
	}

	//Save the signature
	toLicense.Signature = f.Signature
	err = toLicense.SaveSignature(r.Context(), tx)
	if err != nil {
		output.Error(err, "Could not save signature.", w)
		return
	}

	//Commit now to save the license even though we don't know if it is can be
	//successfully validated with the public key.
	err = tx.Commit()
	if err != nil {
		output.Error(err, "Could not complete saving of renewed license.", w)
		return
	}

	//Verify the just created license data and signature. This "writes" out the
	//complete license file with signature and then "reads" it like a third-party app
	//would to verify the signature with a public key. This is done to confirm the
	//signature is valid.
	err = writeReadVerify(f, kp.AlgorithmType, []byte(kp.PublicKey))
	if err == licensefile.ErrBadSignature {
		output.Error(licensefile.ErrBadSignature, "Renewed license could not be verified and therefore cannot be used. Please contact an administrator and have them investigate this error.", w)
		return
	} else if err != nil {
		output.Error(err, "An error occured while trying to verify the license. Please ask an administrator to investigate this error.", w)
		return
	}

	//Mark the license as verified.
	toLicense.Verified = true
	err = toLicense.MarkVerified(r.Context())
	if err != nil {
		output.Error(err, "Could not mark license as valid.", w)
		return
	}

	//Check if user wants the actual license returned. This is typically only for
	//public API requests and is done so that a second request to get the license file
	//isn't needed.
	if r.FormValue("returnLicenseFile") == "true" {
		//Set suggested filename.
		a, err := db.GetAppByID(r.Context(), toLicense.AppID)
		if err != nil {
			output.Error(err, "Could not look up app data to build license filename.", w)
			return
		}

		filename := replaceFilenamePlaceholders(a.DownloadFilename, toLicense.ID, a.Name, a.FileFormat)
		w.Header().Add("Content-Disposition", "inline; filename=\""+filename+"\"")

		err = f.Write(w)
		if err != nil {
			output.Error(err, "Could not return license file.", w)
			return
		}

		//Save download history.
		h := db.DownloadHistory{
			DatetimeCreated:   datetimeCreated,
			TimestampCreated:  time.Now().UnixNano(),
			LicenseID:         toLicense.ID,
			CreatedByUserID:   null.IntFrom(toLicense.CreatedByUserID.Int64),
			CreatedByAPIKeyID: null.IntFrom(toLicense.CreatedByAPIKeyID.Int64),
		}

		err = h.Insert(r.Context())
		if err != nil {
			//not exiting on error since this isn't an end of the world event
			log.Println("license.Renew", "could not save download history", err)
		}

		return
	}

	//Done, renewed license was created and is valid. This will return to the GUI and
	//the GUI will redirect the user to the renewed license's management page.
	output.InsertOK(toLicense.ID, w)
}

// getCreatedBy gets the ID of who/what is creating something. A userID or apiKey will
// be returned, otherwise an error will be returned.
//
// This func was written as a helper for use when creating a license or downloading a
// license file (saving to download history).
func getCreatedBy(r *http.Request) (userID, apiKeyID int64, err error) {
	//Only one of apiKeyID and userID will be provided.
	//  - apiKeyID is set in middleware.ExternalAPI().
	//  - userID is set in middleware.Auth().
	keyID := r.Context().Value(middleware.APIKeyIDCtxKey)
	uID := r.Context().Value(middleware.UserIDCtxKey)

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
// that made a request. See getCreatedBy().
var errUnknownCreatedByID = errors.New("license: unknown creator")

// writeReadVerify is used to verify a just created license data and signature. This
// performs the same "read and verify" that a third-party app would.
func writeReadVerify(f licensefile.File, keyPairAlgo licensefile.KeyPairAlgoType, publicKey []byte) (err error) {
	//Store data that isn't serialized into license file.
	fileFormat := f.FileFormat()

	//Write the license file to a buffer instead of an actual text file or writing
	//to an http response.
	b := bytes.Buffer{}
	err = f.Write(&b)
	if err != nil {
		return
	}

	//"Read" the license file.
	reread, err := licensefile.Unmarshal(b.Bytes(), fileFormat)
	if err != nil {
		return
	}

	//Verify the "reread" license.
	err = reread.VerifySignature(publicKey, keyPairAlgo)
	return
}

// replaceFilenamePlaceholders replaces placeholders in filename that was defined for
// an app with the correct associated data. This generates the actual filename a license
// will be downloaded as.
func replaceFilenamePlaceholders(filename string, licenseID int64, appName string, format licensefile.FileFormat) string {
	filename = strings.ReplaceAll(filename, "{licenseID}", strconv.FormatInt(licenseID, 10))

	filename = strings.ReplaceAll(filename, "{appName}", appName)

	filename = strings.ReplaceAll(filename, "{ext}", string(format))

	return filename
}
