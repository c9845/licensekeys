/*
Package users handles interacting with users of the app.
*/
package users

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/c9845/licensekeys/v3/config"
	"github.com/c9845/licensekeys/v3/db"
	"github.com/c9845/licensekeys/v3/users/pwds"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
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
	loggedInUserID, err := GetUserIDFromRequest(r)
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

	//Check if this user can change this user's password. Admins can change any users'
	//password, but non-admins can only change their own password.
	loggedInUserData, err := GetUserDataFromRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	if !loggedInUserData.Administrator && loggedInUserData.ID != userID {
		output.ErrorInputInvalid("You cannot change the password of another user.", w)
		return
	}

	//Validate.
	if userID <= 0 {
		output.ErrorInputInvalid("Could not determine which user's password you are changing.", w)
		return
	}
	if len(password1) < config.Data().MinPasswordLength {
		output.ErrorInputInvalid("The password you provided is too shSaving new passwordort.  It should be at least "+strconv.Itoa(config.Data().MinPasswordLength)+" characters long.", w)
		return
	}
	if password1 != password2 {
		output.ErrorInputInvalid("Your passwords do not match.", w)
		return
	}

	//Generate password hash.
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

	//Return data if user changed their own password so we can tell user in the GUI
	//that they will be logged out.
	data := struct {
		ChangedOwnPassword bool
	}{}
	if loggedInUserData.ID == userID {
		data.ChangedOwnPassword = true
		DeleteSessionIDCookie(w)
	}

	output.UpdateOKWithData(data, w)
}

// GetOne gets user data for a single user. If no user ID is provided, the data is
// returned for the currently logged in user.
//
// This is used to support the user profile page.
func GetOne(w http.ResponseWriter, r *http.Request) {
	//Get ID of user to look up data for.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)
	if userID < 1 {
		loggedInUserID, err := GetUserIDFromRequest(r)
		if err != nil {
			output.Error(err, "Could not determine the user making this request.", w)
			return
		}

		userID = loggedInUserID
	}

	//Get data.
	columns := sqldb.Columns{
		db.TableUsers + ".*",
	}
	u, err := db.GetUserByID(r.Context(), userID, columns)
	if err != nil {
		output.Error(err, "Could not look up user's data.", w)
		return
	}

	output.DataFound(u, w)
}

// ClearLoginHistory deletes rows in the user logins table before a certain date. This
// is only done from the admin tools page and is done to clean up the database since
// the user login history table can get very big if you have a lot of users and/or a
// short session timeout.
//
// This also clears the user authorized browsers table up to the same data since this
// is tightly related to user logins. This is just easier then making an admin clear
// both tables separately.
//
// The user provides a starting date to delete from, this way you can delete very old
// activity log rows but keep newer history.
func ClearLoginHistory(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	priorToDate := strings.TrimSpace(r.FormValue("priorToDate"))

	//Validate.
	if len(priorToDate) != len("2006-02-02") {
		output.ErrorInputInvalid("Invalid date provided. Date must be in YYYY-MM-DD format and should be a date in the past.", w)
		return
	}

	//Delete.
	rowsDeleted, err := db.ClearUserLogins(r.Context(), priorToDate)
	if err != nil {
		output.Error(err, "Could not clear user login history.", w)
		return
	}

	//Clear authorized browsers, but don't report rows deleted.
	_, err = db.ClearAuthorizedBrowsers(r.Context(), priorToDate)
	if err != nil {
		output.Error(err, "Could not clear authorized browser history for user logins.", w)
		return
	}

	output.UpdateOKWithData(rowsDeleted, w)
}

// LatestLogins retrieves the list of the latest user logins.
func LatestLogins(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)
	rows, _ := strconv.ParseInt(r.FormValue("rows"), 10, 64)

	//Validate. Use defaults if not valid.
	if userID < 0 {
		userID = 0
	}
	if rows < 0 {
		rows = 50
	}

	//Get results.
	logins, err := db.GetUserLogins(r.Context(), userID, uint16(rows))
	if err != nil {
		output.Error(err, "Could not look up list of user logins.", w)
		return
	}

	output.DataFound(logins, w)
}

type userIDContextKeyType string

// UserIDContextKey is the name of the key that stores a user's ID in the request
// context. This is used to save the user ID in middleware-authentication.go and is
// used to get the user ID via context.Value() (or via a helper function below).
const UserIDContextKey userIDContextKeyType = "user-id"

// GetUserIDFromRequest returns the user's ID based on the login ID cookie from the http
// request.
func GetUserIDFromRequest(r *http.Request) (userID int64, err error) {
	id := r.Context().Value(UserIDContextKey)
	if id == nil {
		err = errors.New("could not get user ID from context")
		return
	}

	return id.(int64), nil
}

// GetUserDataFromRequest returns the user's data based on the login cookie from the
// http request.
func GetUserDataFromRequest(r *http.Request) (u db.User, err error) {
	userID, err := GetUserIDFromRequest(r)
	if err != nil {
		return
	}

	u, err = db.GetUserByID(r.Context(), userID, sqldb.Columns{"*"})
	if err != nil {
		return
	}

	return
}
