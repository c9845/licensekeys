/*
Package users handles interacting with users of the app.
*/
package users

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/pwds"
	"github.com/c9845/output"
)

// GetAll gets a list of all users optionally filtered by users that are active.
func GetAll(w http.ResponseWriter, r *http.Request) {
	activeOnly, _ := strconv.ParseBool(r.FormValue("activeOnly"))

	users, err := db.GetUsers(r.Context(), activeOnly)
	if err != nil {
		output.Error(err, "Could not get list of users.", w)
		return
	}

	output.DataFound(users, w)
}

// Add saves a new user.
func Add(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	raw := r.FormValue("data")

	//Parse into struct.
	var u db.User
	err := json.Unmarshal([]byte(raw), &u)
	if err != nil {
		output.Error(err, "Could not parse data to add user.", w)
		return
	}

	//Validate.
	if u.ID != 0 {
		output.ErrorInputInvalid("Could not determine if you are adding or updating a user.", w)
		return
	}

	errMsg, err := u.Validate(r.Context())
	if err != nil && errMsg != "" {
		output.Error(err, errMsg, w)
		return
	} else if err != nil {
		output.Error(err, "Could not validate data about this user.", w)
		return
	} else if errMsg != "" {
		output.ErrorInputInvalid(errMsg, w)
		return
	}

	//Handle password validation.
	if u.PasswordInput1 != u.PasswordInput2 {
		output.ErrorInputInvalid("The passwords you provided do not match.", w)
		return
	}
	if len(u.PasswordInput1) < config.Data().MinPasswordLength {
		output.ErrorInputInvalid("The password you provided is too short. It should be at least "+strconv.Itoa(config.Data().MinPasswordLength)+" characters long.", w)
		return
	}

	//Generate password hash.
	hashedPwd, err := pwds.Create(u.PasswordInput1)
	if err != nil {
		output.Error(err, "Could not add user because of a password issue.", w)
		return
	}
	u.Password = hashedPwd

	//Get user ID of user making this request.
	loggedInUserID, err := GetUserIDByRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	u.CreatedByUserID = loggedInUserID

	//Save.
	err = u.Insert(r.Context())
	if err != nil {
		output.Error(err, "Could not save new user.", w)
		return
	}

	output.InsertOK(u.ID, w)
}

// Update saves changes to a user. This does not handle password changes nor 2 Factor
// Auth stuff since those actions are bit more specialized.
func Update(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	raw := r.FormValue("data")

	//Parse into struct.
	var u db.User
	err := json.Unmarshal([]byte(raw), &u)
	if err != nil {
		output.Error(err, "Could not parse data to update user.", w)
		return
	}

	//Validate.
	if u.ID < 1 {
		output.ErrorInputInvalid("Could not determine which user you are updating", w)
		return
	}

	errMsg, err := u.Validate(r.Context())
	if err != nil && errMsg != "" {
		output.Error(err, errMsg, w)
		return
	} else if err != nil {
		output.Error(err, "Could not validate data about this user.", w)
		return
	} else if errMsg != "" {
		output.ErrorInputInvalid(errMsg, w)
		return
	}

	//Save.
	err = u.Update(r.Context())
	if err != nil {
		output.Error(err, "Could not update user.", w)
		return
	}

	output.UpdateOK(w)
}

// ChangePassword sets a new password for a user
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)
	password1 := r.FormValue("password1")
	password2 := r.FormValue("password2")

	//Validate.
	if userID <= 0 {
		output.ErrorInputInvalid("Could not determine which user's password you are changing.", w)
		return
	}
	if len(password1) < config.Data().MinPasswordLength {
		output.ErrorInputInvalid("The password you provided is too short.  It should be at least "+strconv.Itoa(config.Data().MinPasswordLength)+" characters long.", w)
		return
	}
	if password1 != password2 {
		output.ErrorInputInvalid("Your passwords do not match.", w)
		return
	}

	//Generate password.
	hashedPwd, err := pwds.Create(password1)
	if err != nil {
		output.Error(err, "Could not add user because of a password issue.", w)
		return
	}

	//Save.
	err = db.SetNewPassword(r.Context(), userID, hashedPwd)
	if err != nil {
		output.Error(err, "Could not set new password.", w)
		return
	}

	//Inactivate all existing active user logins/sessions for security.
	err = db.DisableLoginsForUser(r.Context(), userID)
	if err != nil {
		//not exiting on this error since it isn't a huge issue.
		log.Println("users.ChangePassword", "could not disable logins for user", err)
	}

	output.UpdateOK(w)
}
