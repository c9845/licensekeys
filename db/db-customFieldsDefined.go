package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/c9845/licensekeys/v2/timestamps"
	"github.com/c9845/sqldb/v3"
	"golang.org/x/exp/slices"
	"gopkg.in/guregu/null.v3"
)

//This table stores the extra custom fields defined for an app. Custom fields
//allow you to store arbitrary data in a license such as max user count, flags
//to enable certain features, etc. An app can have as many custom fields as you
//want, however, custom fields are not required. This table is populated for each
//app and is used each time a user views the gui to create a new license.
//
//Custom fields are encoded into the "Extra" field of each license as a key/value
//pair if at least one custom field is provided. The Extra field is taken into
//account when the license is signed so that end-users cannot modify the values
//of any custom fields.

// TableCustomFieldDefined is the name of the table.
const TableCustomFieldDefined = "custom_fields_defined"

// CustomFieldDefined is used to interact with the table.
type CustomFieldDefined struct {
	ID               int64
	DatetimeCreated  string
	DatetimeModified string
	CreatedByUserID  int64
	Active           bool

	AppID        int64           //what app this field is for.
	Type         customFieldType //field type
	Name         string          //
	Instructions string          //what field is for and expected value

	//defaults to populate GUI when user is creating a new key
	IntegerDefaultValue     null.Int
	DecimalDefaultValue     null.Float
	TextDefaultValue        null.String
	BoolDefaultValue        null.Bool
	MultiChoiceDefaultValue null.String
	DateDefaultIncrement    null.Int //number of days incremented from "today", date license is being created

	//validation for value chosen/given in gui
	NumberMinValue     null.Float  //for integers and decimals
	NumberMaxValue     null.Float  //""
	MultiChoiceOptions null.String //semicolon separated list of options

	//When saving a license, we retrieve the defined fields for an app
	//and set the value for each field using the same list of objects
	//returned just for ease of use and not changing types. Therefore,
	//we need fields in this struct to store the chosen/provided
	//values for each field. Only one of these fields is populated per
	//the Type field. These are only populated when saving/creating a
	//new license.
	IntegerValue     int64   `json:",omitempty"`
	Decimalvalue     float64 `json:",omitempty"`
	TextValue        string  `json:",omitempty"`
	BoolValue        bool    `json:",omitempty"`
	MultiChoiceValue string  `json:",omitempty"`
	DateValue        string  `json:",omitempty"`
}

const (
	createTableCustomFieldsDefined = `
		CREATE TABLE IF NOT EXISTS ` + TableCustomFieldDefined + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified TEXT DEFAULT CURRENT_TIMESTAMP,
			CreatedByUserID INTEGER NOT NULL,
			Active INTEGER NOT NULL DEFAULT 1,

			AppID INTEGER NOT NULL,
			Type TEXT NOT NULL,
			Name TEXT NOT NULL,
			Instructions TEXT NOT NULL DEFAULT '',
			
			IntegerDefaultValue INTEGER DEFAULT NULL,
			DecimalDefaultValue REAL DEFAULT NULL,
			TextDefaultValue TEXT DEFAULT NULL,
			BoolDefaultValue INTEGER DEFAULT NULL,
			MultiChoiceDefaultValue TEXT DEFAULT NULL,
			DateDefaultIncrement TEXT DEFAULT NULL,
			
			NumberMinValue REAL DEFAULT NULL,
			NumberMaxValue REAL DEFAULT NULL,
			MultiChoiceOptions TEXT DEFAULT NULL,

			FOREIGN KEY (CreatedByUserID) REFERENCES ` + TableUsers + `(ID),
			FOREIGN KEY (AppID) REFERENCES ` + TableApps + `(ID)
		)
	`
)

// Define the types of custom fields this app supports.
type customFieldType string

const (
	CustomFieldTypeInteger     = customFieldType("Integer")
	CustomFieldTypeDecimal     = customFieldType("Decimal")
	CustomFieldTypeText        = customFieldType("Text")
	CustomFieldTypeBoolean     = customFieldType("Boolean")
	CustomFieldTypeMultiChoice = customFieldType("Multi-Choice")
	CustomFieldTypeDate        = customFieldType("Date")
)

var customFieldTypes = []customFieldType{
	CustomFieldTypeInteger,
	CustomFieldTypeDecimal,
	CustomFieldTypeText,
	CustomFieldTypeBoolean,
	CustomFieldTypeMultiChoice,
	CustomFieldTypeDate,
}

// multiSeparator is the character used to split options for a multichoice field when
// provided as a single string. Ex: ["a", "b", "c"] -> "a;b;c".
// This is exported for use when we validate the option chosen when a user
// creates/saves a license.
const multiSeparator = ";"

// Valid checks if a provided field type is one of our supported types. This is used
// for validation purposes.
func (c customFieldType) Valid() bool {
	for _, v := range customFieldTypes {
		if v == c {
			return true
		}
	}

	return false
}

// Validate is used to validate a struct's data before adding or saving changes. This also
// handles sanitizing.
func (cfd *CustomFieldDefined) Validate(ctx context.Context) (errMsg string, err error) {
	//Sanitize
	cfd.Name = strings.TrimSpace(cfd.Name)
	cfd.Instructions = strings.TrimSpace(cfd.Instructions)

	//Validate
	if cfd.Name == "" {
		errMsg = "You must provide a name for this field."
		return
	}
	if cfd.AppID < 1 {
		errMsg = "Could not determine which app you are adding a field for. Please refresh and try again."
		return
	}

	switch cfd.Type {
	case CustomFieldTypeInteger:
		if cfd.NumberMinValue.Float64 >= cfd.NumberMaxValue.Float64 {
			errMsg = "The minimum value must be less than the maximum"
			return
		}

	case CustomFieldTypeDecimal:
		if cfd.DecimalDefaultValue.Float64 < cfd.NumberMinValue.Float64 || cfd.DecimalDefaultValue.Float64 > cfd.NumberMaxValue.Float64 {
			errMsg = "The default value must be within the minimum to maximum range."
			return
		}

	case CustomFieldTypeText:
		if cfd.TextDefaultValue.String == "" {
			errMsg = "You must choose a default value for this field."
			return
		}

	case CustomFieldTypeBoolean:
		//nothing to do here

	case CustomFieldTypeMultiChoice:
		//make sure some options were given
		if cfd.MultiChoiceOptions.String == "" {
			errMsg = "You must provide at least one option."
			return
		}

		//When validating multi choice options, we handle the provided options
		//as an array. This allows for easier sanitizing and validating since we
		//are going to display the options in a select from an array anyway.
		opsArray := strings.Split(cfd.MultiChoiceOptions.String, multiSeparator)

		//remove starting/ending whitespace from each option
		//only put non-blank results into output array
		//make sure element doesn't already exist in output array, warning if duplicate is found.
		validatedArray := []string{}
		for _, v := range opsArray {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}

			if slices.Contains(validatedArray, v) {
				errMsg = "The option \"" + v + "\" is included more than once. Please remove the duplicate(s)."
				return
			}

			validatedArray = append(validatedArray, v)
		}

		//make sure something exists in the array.
		if len(validatedArray) == 0 {
			errMsg = "You must provide as least one option for this field."
			return
		}

		//make sure default value is in list of options.
		cfd.MultiChoiceDefaultValue.String = strings.TrimSpace(cfd.MultiChoiceDefaultValue.String)
		if cfd.MultiChoiceDefaultValue.String == "" {
			errMsg = "You must choose a default value for this field."
			return
		}
		if !slices.Contains(validatedArray, cfd.MultiChoiceDefaultValue.String) {
			errMsg = "Please choose a default value from the list of options your provided."
			return
		}

		//set validated array back for saving to db as a separated string
		cfd.MultiChoiceOptions.String = strings.Join(validatedArray, multiSeparator)

	case CustomFieldTypeDate:
	default:
		errMsg = "Please choose an field type from the provided options."
		return
	}

	//Check if a field with this name already exists for this app an is active. We don't
	//want duplicate field names.
	existing, err := GetFieldByName(ctx, cfd.AppID, cfd.Name)
	if err == sql.ErrNoRows {
		err = nil
	} else if err != nil {
		//some kind of db error occured
		return
	} else if (err == nil) && ((cfd.ID > 0 && cfd.ID != existing.ID) || (cfd.ID == 0)) {
		//no db error occured, but an existing field was returned. We have to determine
		//if "this" field was returned and we are ok since we are updating it and therefore
		//no duplicate name will result.
		errMsg = "A field with this name already exists for this app."
		return
	}

	return
}

// GetFieldByName looks up a field by its name for a given app. We filter by app since
// multiple apps can have the same fields names.
func GetFieldByName(ctx context.Context, appID int64, name string) (cfd CustomFieldDefined, err error) {
	q := `
		SELECT ` + TableCustomFieldDefined + `.*
		FROM ` + TableCustomFieldDefined + `
		WHERE 
			(Name = ?)
			AND
			(AppID = ?)
	`

	c := sqldb.Connection()
	err = c.GetContext(ctx, &cfd, q, name, appID)
	return
}

// Insert saves a defined field. You should have already called Validate().
func (cfd *CustomFieldDefined) Insert(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"CreatedByUserID",
		"Active",
		"AppID",
		"Type",
		"Name",
		"Instructions",
	}
	b := sqldb.Bindvars{
		cfd.CreatedByUserID,
		cfd.Active,
		cfd.AppID,
		cfd.Type,
		cfd.Name,
		cfd.Instructions,
	}

	switch cfd.Type {
	case CustomFieldTypeInteger:
		cols = append(cols, "IntegerDefaultValue", "NumberMinValue", "NumberMaxValue")
		b = append(b, cfd.IntegerDefaultValue.Int64, cfd.NumberMinValue.Float64, cfd.NumberMaxValue.Float64)

	case CustomFieldTypeDecimal:
		cols = append(cols, "DecimalDefaultValue", "NumberMinValue", "NumberMaxValue")
		b = append(b, cfd.DecimalDefaultValue.Float64, cfd.NumberMinValue.Float64, cfd.NumberMaxValue.Float64)

	case CustomFieldTypeText:
		cols = append(cols, "TextDefaultValue")
		b = append(b, cfd.TextDefaultValue)

	case CustomFieldTypeBoolean:
		cols = append(cols, "BoolDefaultValue")
		b = append(b, cfd.BoolDefaultValue)

	case CustomFieldTypeMultiChoice:
		cols = append(cols, "MultiChoiceDefaultValue", "MultiChoiceOptions")
		b = append(b, cfd.MultiChoiceDefaultValue, cfd.MultiChoiceOptions)

	case CustomFieldTypeDate:
		cols = append(cols, "DateDefaultIncrement")
		b = append(b, cfd.DateDefaultIncrement)

	default:
		err = errors.New("unknown custom field type, this should have been caught by Validate() " + string(cfd.Type))
		return
	}

	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableCustomFieldDefined + `(` + colString + `) VALUES (` + valString + `)`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, b...)
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	cfd.ID = id
	return
}

// GetCustomFieldsDefined returns the list of fields for an app optionally filtered by active fields only.
func GetCustomFieldsDefined(ctx context.Context, appID int64, activeOnly bool) (cc []CustomFieldDefined, err error) {
	//base query
	q := `
		SELECT ` + TableCustomFieldDefined + `.* 
		FROM ` + TableCustomFieldDefined

	//filters
	wheres := []string{}
	b := sqldb.Bindvars{}

	w := `(` + TableCustomFieldDefined + `.AppID = ?)`
	wheres = append(wheres, w)
	b = append(b, appID)

	if activeOnly {
		w := `(` + TableCustomFieldDefined + `.Active = ?)`
		wheres = append(wheres, w)
		b = append(b, activeOnly)
	}

	if len(wheres) > 0 {
		where := " WHERE " + strings.Join(wheres, " AND ")
		q += where
	}

	//complete query
	q += ` ORDER BY ` + TableCustomFieldDefined + `.Active DESC, ` + TableCustomFieldDefined + `.Name COLLATE NOCASE ASC`

	//run query
	c := sqldb.Connection()
	err = c.SelectContext(ctx, &cc, q, b...)
	return
}

// Update saves changes to a defined field. You should have already called Validate().
func (cfd *CustomFieldDefined) Update(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"DatetimeModified",
		"Name",
		"Instructions",
	}
	b := sqldb.Bindvars{
		timestamps.YMDHMS(),
		cfd.Name,
		cfd.Instructions,
	}

	switch cfd.Type {
	case CustomFieldTypeInteger:
		cols = append(cols, "IntegerDefaultValue", "NumberMinValue", "NumberMaxValue")
		b = append(b, cfd.IntegerDefaultValue.Int64, cfd.NumberMinValue.Float64, cfd.NumberMaxValue.Float64)

	case CustomFieldTypeDecimal:
		cols = append(cols, "DecimalDefaultValue", "NumberMinValue", "NumberMaxValue")
		b = append(b, cfd.DecimalDefaultValue.Float64, cfd.NumberMinValue.Float64, cfd.NumberMaxValue.Float64)

	case CustomFieldTypeText:
		cols = append(cols, "TextDefaultValue")
		b = append(b, cfd.TextDefaultValue)

	case CustomFieldTypeBoolean:
		cols = append(cols, "BoolDefaultValue")
		b = append(b, cfd.BoolDefaultValue)

	case CustomFieldTypeMultiChoice:
		cols = append(cols, "MultiChoiceDefaultValue", "MultiChoiceOptions")
		b = append(b, cfd.MultiChoiceDefaultValue, cfd.MultiChoiceOptions)

	case CustomFieldTypeDate:
		cols = append(cols, "DateDefaultIncrement")
		b = append(b, cfd.DateDefaultIncrement)

	default:
		err = errors.New("unknown custom field type, this should have been caught by Validate() " + string(cfd.Type))
		return
	}

	colString, err := cols.ForUpdate()
	if err != nil {
		return
	}

	//add field id to bindvar for WHERE
	b = append(b, cfd.ID)

	q := `UPDATE ` + TableCustomFieldDefined + ` SET ` + colString + ` WHERE ID = ?`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		b...,
	)
	return
}

// Delete marks a defined custom field as deleted.
func (cfd *CustomFieldDefined) Delete(ctx context.Context) (err error) {
	q := `
		UPDATE ` + TableCustomFieldDefined + ` 
		SET 
			Active = ?,
			DatetimeModified = ?
		WHERE ID = ?
	`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		false,
		timestamps.YMDHMS(),

		cfd.ID,
	)
	return
}
