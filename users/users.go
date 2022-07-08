/*
Package users handles interacting with users of the app.
*/
package users

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/c9845/licensekeys/config"
	"github.com/c9845/licensekeys/db"
	"github.com/c9845/licensekeys/pwds"
	"github.com/c9845/output"
)

//GetAll gets a list of all users optionally filtered by users that are active.
func GetAll(w http.ResponseWriter, r *http.Request) {
	activeOnly, _ := strconv.ParseBool(r.FormValue("activeOnly"))

	users, err := db.GetUsers(r.Context(), activeOnly)
	if err != nil {
		output.Error(err, "Could not get list of users.", w)
		return
	}

	output.DataFound(users, w)
}

//Add saves a new user.
func Add(w http.ResponseWriter, r *http.Request) {
	//get input data
	//should be a json payload matching the db.User struct
	raw := r.FormValue("data")

	//parse into struct
	var u db.User
	err := json.Unmarshal([]byte(raw), &u)
	if err != nil {
		output.Error(err, "Could not parse data to add user.", w)
		return
	}

	//validate and sanitize
	if u.ID != 0 {
		output.ErrorInputInvalid("Could not determine if you are adding or updating a user. Please refresh and try again.", w)
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

	//handle password validation
	if u.PasswordInput1 != u.PasswordInput2 {
		output.ErrorInputInvalid("The passwords you provided do not match.", w)
		return
	}
	if len(u.PasswordInput1) < config.Data().MinPasswordLength {
		output.ErrorInputInvalid("The password you provided is too short. It should be at least "+strconv.Itoa(config.Data().MinPasswordLength)+" characters long.", w)
		return
	}

	//generate password hash
	hashedPwd, err := pwds.Create(u.PasswordInput1)
	if err != nil {
		output.Error(err, "Could not add user because of a password issue.", w)
		return
	}
	u.Password = hashedPwd

	//get user id of logged in user
	loggedInUserID, err := GetUserIDByRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	u.CreatedByUserID = loggedInUserID

	//save
	err = u.Insert(r.Context())
	if err != nil {
		output.Error(err, "Could not save new user.", w)
		return
	}

	output.InsertOK(u.ID, w)
}

//Update saves changes to a user
func Update(w http.ResponseWriter, r *http.Request) {
	//get input data
	//should be a json payload matching the db.User struct
	raw := r.FormValue("data")

	//parse into struct
	var u db.User
	err := json.Unmarshal([]byte(raw), &u)
	if err != nil {
		output.Error(err, "Could not parse data to update user.", w)
		return
	}

	//validate and sanitize
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

	//save the user
	err = u.Update(r.Context())
	if err != nil {
		output.Error(err, "Could not update user.", w)
		return
	}

	output.UpdateOK(w)
}

//ChangePassword sets a new password for a user
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	//get inputs
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)
	password1 := r.FormValue("password1")
	password2 := r.FormValue("password2")

	//validate
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

	//generate password
	hashedPwd, err := pwds.Create(password1)
	if err != nil {
		output.Error(err, "Could not add user because of a password issue.", w)
		return
	}

	//save the new password
	err = db.SetNewPassword(r.Context(), userID, hashedPwd)
	if err != nil {
		output.Error(err, "Could not set new password.", w)
		return
	}

	//inactivate all existing active user logins/sessions for security
	err = db.DisableLoginsForUser(r.Context(), userID)
	if err != nil {
		//not exiting on this error since it isn't a huge issue.
		log.Println("users.ChangePassword", "could not disable logins for user", err)
	}

	//done
	output.UpdateOK(w)
}
