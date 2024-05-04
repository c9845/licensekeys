package db

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/sqldb/v3"
	"github.com/jmoiron/sqlx"
	"golang.org/x/exp/slices"
	"gopkg.in/guregu/null.v3"
)

//This table stores the chosen values of each custom field for a specificly generated
//license. This data is used so that you can always go back and view the details of
//a license when it was generated or redownload a license if needed. This table is
//populated each time a new license is created and is used whenever a license needs
//to be viewed or downloaded.

// TableCustomFieldResults is the name of the table.
const TableCustomFieldResults = "custom_field_results"

// CustomFieldResult is used to interact with the table.
type CustomFieldResult struct {
	ID              int64
	DatetimeCreated string

	//a license can be created by a user or via an api call.
	CreatedByUserID   null.Int
	CreatedByAPIKeyID null.Int

	LicenseID            int64           //what license this result was set for
	CustomFieldDefinedID int64           //what field this result is related to
	CustomFieldType      customFieldType //field type, since type could be changed on defined field
	CustomFieldName      string          //name of field when result was set, since name could be changed on defined field

	IntegerValue     null.Int    //provided result value for an integer field
	DecimalValue     null.Float  //provided result value for a decminal field.
	TextValue        null.String //provided result value for a text field
	BoolValue        null.Bool   //provided result for a boolean field
	MultiChoiceValue null.String //chosen result for a multi choice field
	DateValue        null.String //provided result for a date field
}

// MultiCustomFieldResult is used so that we can defined a method on a
// slice type. You cannot do func (c []CustomField) FuncName...
type MultiCustomFieldResult []CustomFieldResult

const (
	createTableCustomFieldResults = `
		CREATE TABLE IF NOT EXISTS ` + TableCustomFieldResults + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			
			CreatedByUserID INTEGER DEFAULT NULL,
			CreatedByAPIKeyID INTEGER DEFAULT NULL,

			LicenseID INTEGER NOT NULL,
			CustomFieldDefinedID INTEGER NOT NULL,
			CustomFieldType TEXT NOT NULL,
			CustomFieldName TEXT NOT NULL,
			IntegerValue INTEGER DEFAULT NULL,
			DecimalValue REAL DEFAULT NULL,
			TextValue TEXT DEFAULT NULL,
			BoolValue INTEGER DEFAULT NULL,
			MultiChoiceValue TEXT DEFAULT NULL,
			DateValue TEXT DEFAULT NULL,

			FOREIGN KEY (CreatedByUserID) REFERENCES ` + TableUsers + `(ID),
			FOREIGN KEY (CreatedByAPIKeyID) REFERENCES ` + TableAPIKeys + `(ID),
			FOREIGN KEY (LicenseID) REFERENCES ` + TableLicenses + `(ID)
		)
	`
)

// Insert saves an app. You should have already called Validate().
func (f *CustomFieldResult) Insert(ctx context.Context, tx *sqlx.Tx) (err error) {
	cols := sqldb.Columns{
		"DatetimeCreated",
		"LicenseID",
		"CustomFieldDefinedID",
		"CustomFieldType",
		"CustomFieldName",
	}
	b := sqldb.Bindvars{
		f.DatetimeCreated,
		f.LicenseID,
		f.CustomFieldDefinedID,
		f.CustomFieldType,
		f.CustomFieldName,
	}

	if f.CreatedByUserID.Int64 > 0 {
		cols = append(cols, "CreatedByUserID")
		b = append(b, f.CreatedByUserID.Int64)
	} else {
		cols = append(cols, "CreatedByAPIKeyID")
		b = append(b, f.CreatedByAPIKeyID.Int64)
	}

	switch f.CustomFieldType {
	case CustomFieldTypeInteger:
		cols = append(cols, "IntegerValue")
		b = append(b, f.IntegerValue)
	case CustomFieldTypeDecimal:
		cols = append(cols, "DecimalValue")
		b = append(b, f.DecimalValue)
	case CustomFieldTypeText:
		cols = append(cols, "TextValue")
		b = append(b, f.TextValue)
	case CustomFieldTypeBoolean:
		cols = append(cols, "BoolValue")
		b = append(b, f.BoolValue)
	case CustomFieldTypeMultiChoice:
		cols = append(cols, "MultiChoiceValue")
		b = append(b, f.MultiChoiceValue)
	case CustomFieldTypeDate:
		cols = append(cols, "DateValue")
		b = append(b, f.DateValue)
	default:
		//you should have already performed validation so this should never be hit.
	}

	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableCustomFieldResults + `(` + colString + `) VALUES (` + valString + `)`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, b...)
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	f.ID = id
	return
}

// GetCustomFieldResults looks up the list of saved values for a license.
func GetCustomFieldResults(ctx context.Context, licenseID int64) (ff []CustomFieldResult, err error) {
	q := `
		SELECT ` + TableCustomFieldResults + `.*
		FROM ` + TableCustomFieldResults + ` 
		WHERE ` + TableCustomFieldResults + `.LicenseID = ?
		ORDER BY ` + TableCustomFieldResults + `.CustomFieldName COLLATE NOCASE ASC
	`

	c := sqldb.Connection()
	err = c.SelectContext(ctx, &ff, q, licenseID)
	return
}

// Validate handles validating results for each custom field provided when creating
// a new license.
//
// An error can be returned as a string, errMsg, or an error, err. This is done
// because if an input validation error occurs we will not have an error type value
// to pass back, but we will have a human-readable message to display to the user.
//
// The input provided results are modified with data from the defined field to use
// when saving the results to the database.
func (results MultiCustomFieldResult) Validate(ctx context.Context, appID int64, viaAPI bool) (errMsg string, err error) {
	//Look up fields defined for app.
	definedFields, err := GetCustomFieldsDefined(ctx, appID, true)
	if err != nil {
		return
	}

	//Check if count of defined and result fields matches. This is just a simply
	//validation step to see if not enough or too many results were provided.
	if len(definedFields) != len(results) {
		errMsg = "The number of results provided does not match what is expected."
		err = errors.New("number of results does not match")
		return
	}

	//Validate each result provided. This loops through each defined field and
	//looks for a matching result field. If a match is not found, then clearly the
	//results provided are incorrect. If a match is found, we then check if the
	//value for the field is valid for the field type.
	for _, definedField := range definedFields {
		//Find a matching user provided result. This works by matching up IDs. When
		//the GUI is populated with the fields to fill out, the API call looks up
		//the defined fields for the app. Therefore the defined field's ID is
		//present. When the data is submitted to the server for saving, the defined
		//field's ID is "moved" to the CustomFieldDefinedID field as would be
		//expected for a result object. The server parses the submitted data as
		//"results" so the ID is in the correct place.
		//
		//Note that we match the result based on the field's name when user is adding
		//a license via an API call. This is allowed since building an API call with
		//field names is easier then with field IDs (IDs aren't even public!).
		var matchingResult CustomFieldResult
		var matchingFieldIndex int
		var matchingFound bool
		for idx, result := range results {
			if result.CustomFieldDefinedID == definedField.ID {
				matchingResult = result
				matchingFieldIndex = idx
				matchingFound = true
				break
			} else if viaAPI && result.CustomFieldName == definedField.Name {
				matchingResult = result
				matchingFieldIndex = idx
				matchingFound = true
				break
			}
		}

		//Make sure we found a matching result for the defined field.
		if !matchingFound {
			errMsg = "Could not find matching result for field " + definedField.Name + "."
			err = ErrCouldNotFindMatchingField
			return
		}

		//Set some data for the result from data from the database. Note that some of
		//these fields will already be set because they were retrieved to build the GUI
		//and then submitted along with the result to the server when saving. We just
		//reset the values here to make sure they are correct from what is set for the
		//defined field since we can trust the defined field's data (we cannot trust data
		//from the GUI).
		matchingResult.ID = 0 //make sure none is already set.
		matchingResult.CustomFieldDefinedID = definedField.ID
		matchingResult.CustomFieldType = definedField.Type
		matchingResult.CustomFieldName = definedField.Name

		//Unset fields that we set when we looked up defined fields.
		matchingResult.CreatedByUserID = null.IntFrom(0)
		matchingResult.DatetimeCreated = ""

		//Validate the matching field's value that was provided in the GUI. This
		//will also set more data in the matchingResult to be saved to the database.
		switch definedField.Type {
		case CustomFieldTypeInteger:
			if matchingResult.IntegerValue.Int64 < int64(definedField.NumberMinValue.Float64) || matchingResult.IntegerValue.Int64 > int64(definedField.NumberMaxValue.Float64) {
				errMsg = "The value for the " + definedField.Name + " field must be an integer between " + strconv.FormatInt(int64(definedField.NumberMinValue.Float64), 10) + " and " + strconv.FormatInt(int64(definedField.NumberMaxValue.Float64), 10) + "."
				return
			}

		case CustomFieldTypeDecimal:
			if matchingResult.DecimalValue.Float64 < definedField.NumberMinValue.Float64 || matchingResult.DecimalValue.Float64 > definedField.NumberMaxValue.Float64 {
				errMsg = "The value for the " + definedField.Name + " field must be a decimal between " + strconv.FormatFloat(definedField.NumberMinValue.Float64, 'f', 2, 64) + " and " + strconv.FormatFloat(definedField.NumberMaxValue.Float64, 'f', 2, 64) + "."
				return
			}

		case CustomFieldTypeText:
			//blank values are acceptable for text fields
			matchingResult.TextValue = null.StringFrom(strings.TrimSpace(matchingResult.TextValue.String))

		case CustomFieldTypeBoolean:
			//default to false

		case CustomFieldTypeMultiChoice:
			ops := strings.Split(definedField.MultiChoiceOptions.String, multiSeparator)
			if !slices.Contains(ops, matchingResult.MultiChoiceValue.String) {
				errMsg = "You must choose a value for the " + definedField.Name + " field."
				return
			}

		case CustomFieldTypeDate:
			//Make sure a valid date string was provided and that it is in the future.
			//It wouldn't make sense if this was blank or in the past.
			matchingResult.DateValue = null.StringFrom(strings.TrimSpace(matchingResult.DateValue.String))
			if matchingResult.DateValue.String == "" {
				errMsg = "You must provide a date for the " + definedField.Name + " field."
				return
			}
			t, innerErr := time.Parse("2006-01-02", matchingResult.DateValue.String)
			if innerErr != nil {
				err = innerErr
				return
			}
			if !t.After(time.Now()) {
				errMsg = "The date for the " + definedField.Name + " field must be in the future."
				return
			}

		default:
			//This should never occur since we are looping through defined custom
			//fields retrieved from the database and we validated the type of each
			//field when we saved it to the db.
		} //end switch: validate based on field type

		//Update the list of provided result fields since we added data from the
		//matching defined field.
		results[matchingFieldIndex] = matchingResult

	} //end for: loop through defined fields for app and validating matching values

	return
}
