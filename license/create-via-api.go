package license

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v4/apikeys"
	"github.com/c9845/licensekeys/v4/db"
	"github.com/c9845/output"
	"gopkg.in/guregu/null.v3"
)

// CreateViaPublicAPI creates a license via the public API.
//
// The request can specify EITHER a keypair ID or an app ID, but not both. If an app
// ID is provided, the default keypair for that app is used.
//
// When data is provided via the public API, custom fields are provided as an object
// of key-value pairs (custom field name to value). This just makes providing data
// easier since the request doesn't have to build an array of objects with
// "name:xxx", "value:yyy" pairings like the GUI request does. The key, field name,
// *must* match exactly to the field name shown in the app's GUI.
func CreateViaPublicAPI(w http.ResponseWriter, r *http.Request) {
	//Determine which app and keypair to use.
	//
	//If user provided app and keypair, tell them! Don't just use one over the other
	//since users will not expect this. Make the user fix the mistake.
	appPublicID := strings.TrimSpace(r.FormValue("appID"))         //UUID
	keypairPublicID := strings.TrimSpace(r.FormValue("keypairID")) //UUID
	if appPublicID != "" && keypairPublicID != "" {
		output.ErrorInputInvalid("Please provide either an appID or keypairID, not both.", w)
		return
	}
	if appPublicID == "" && keypairPublicID == "" {
		output.ErrorInputInvalid("Missing appID and keypairID. You must provide a value for one of these fields.", w)
		return
	}

	//If user provided an app ID, look up the default keypair.
	//If user provided a keypair ID, look up the related app.
	var keypairID int64
	var appID int64
	if appPublicID != "" {
		kp, err := db.GetDefaultKeypairByAppPublicID(r.Context(), db.UUID(appPublicID))
		if err != nil {
			output.Error(err, "Could not look up default keypair for provided appID.", w)
			return
		}
		keypairID = kp.ID

	} else if keypairPublicID != "" {
		kp, err := db.GetKeypairByID(r.Context(), keypairID)
		if err != nil {
			output.Error(err, "Could not look up app based on provided keypairID.", w)
			return
		}
		appID = kp.AppID
	}

	//Parse and validate basic license file data from request.
	l := db.License{
		AppID:          appID,
		KeyPairID:      keypairID,
		CompanyName:    r.FormValue("companyName"),
		ContactName:    r.FormValue("contactName"),
		PhoneNumber:    r.FormValue("phoneNumber"),
		Email:          r.FormValue("email"),
		ExpirationDate: r.FormValue("expirationDate"),
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
	app, err := db.GetAppByID(r.Context(), appID)
	if err != nil {
		output.Error(err, "Could not look up app.", w)
		return
	}
	if !app.Active {
		output.ErrorInputInvalid("This app is not active, you cannot create licenses for it.", w)
		return
	}

	//Look up and validate keypair.
	kp, err := db.GetKeypairByID(r.Context(), l.KeyPairID)
	if err != nil {
		output.Error(err, "Could not look up keypair.", w)
		return
	}
	if !kp.Active {
		output.ErrorInputInvalid("This keypair is not active, you cannot create licenses with it.", w)
		return
	}

	//Parse custom fields from request.
	//
	//Custom fields are provided as an object of key-value pairs. The key is a custom
	//field's name, and the value is the field's value. The key/field-name, must match
	//exactly to the field shown in this app's GUI so we can match up fields properly.
	rawObject, err := url.QueryUnescape(r.FormValue("customFields"))
	if err != nil {
		output.Error(err, "Could not parse custom fields.", w)
		return
	}

	var requestObj map[string]interface{}
	err = json.Unmarshal([]byte(rawObject), &requestObj)
	if err != nil && len(rawObject) > 0 {
		//We check the length to handle times when the request didn't provide any
		//custom field data. In this case, we can ignore the json unmarshal error
		//since we will just use the default value for every custom field.
		output.Error(err, "Could not parse custom fields.", w)
		return
	}

	//Look up custom fields defined for app so that we can match up against the fields
	//provided, perform validation, and to get correct type for each field to build
	//slice correctly.
	definedFields, err := db.GetCustomFieldsDefined(r.Context(), l.AppID, true)
	if err != nil {
		output.Error(err, "Could not look up custom fields.", w)
		return
	}

	//Loop through each defined custom field, trying to find a matching value provided
	//in the request. If a match is found, build a "result" struct with the data.
	//
	//This basically translates the user-provided object into a slice of structs that
	//this app uses internally.
	var cfr []db.CustomFieldResult
	for _, definedField := range definedFields {
		//Build the base object for the field to be saved.
		c := db.CustomFieldResult{
			CustomFieldName:      definedField.Name,
			CustomFieldDefinedID: definedField.ID,
		}

		//Find the matching value provided in the request.
		matchFound := false
		for key, value := range requestObj {
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
					log.Println("license.CreateViaPublicAPI", "invalid field type, this should never happen", definedField.Type, key, value)
				}
			}
		} //end for: find matching value for field name.

		//Handle if no matching field was provided. Most likely the field was not
		//provided in the request or the field name was spelled incorrectly. In this
		//case, we will just use the default value set for the field.
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
				//
				//No logging, this should have been caught above.
			}

			// log.Println("license.AddViaAPI", "using default", definedField.Name)
		} //end if: use default value for not provided field

		cfr = append(cfr, c)
	} //end for: loop through defined fields

	//Get API key that is creating this license.
	keyID := r.Context().Value(apikeys.APIKeyContextKey)
	l.CreatedByAPIKeyID = null.IntFrom(keyID.(int64))

	//Create and save license.
	licenseID, file, err := create(r.Context(), l, cfr, app, kp, 0)
	if err != nil {
		output.Error(err, "Could not create license.", w)
		return
	}

	//Check if user wants the actual license file returned. This is typically done so
	//that a second request to get the license file isn't needed.
	if r.FormValue("returnLicenseFile") == "true" {
		//Set filename for downloading.
		w.Header().Add("Content-Disposition", "inline; filename=\""+app.DownloadFilename+"\"")

		//Return the file.
		err = file.Write(w)
		if err != nil {
			output.Error(err, "Could not return license file.", w)
			return
		}

		//Save download history.
		h := db.DownloadHistory{
			TimestampCreated:  time.Now().UnixNano(),
			LicenseID:         licenseID,
			CreatedByUserID:   null.IntFrom(l.CreatedByUserID.Int64),
			CreatedByAPIKeyID: null.IntFrom(l.CreatedByAPIKeyID.Int64),
		}

		err = h.Insert(r.Context())
		if err != nil {
			//not exiting on error since this isn't an end of the world event
			log.Println("license.CreateViaPublicAPI", "could not save download history", err)
		}

		return

	}

	output.InsertOK(licenseID, w)
}
