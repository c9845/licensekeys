package users

import (
	"net/http"
	"strconv"

	"github.com/c9845/licensekeys/v3/config"
	"github.com/c9845/licensekeys/v3/db"
	"github.com/c9845/output"
)

// Logout handles logging a user out.
// Remove the session info so users isn't automatically logged back in to the app.
// Remove the 2FA token if config requires 2FA upon each login.
func Logout(w http.ResponseWriter, r *http.Request) {
	DeleteSessionIDCookie(w)

	if config.Data().TwoFactorAuthLifetimeDays < 0 {
		Delete2FABrowserIDCookie(w)
	}

	http.Redirect(w, r, "/?ref=logout", http.StatusFound)
}

// ForceLogout handles requests to force a user to log out of the app. This invalidates
// all non-expired, active user logins causing all subsequent requests (page views or api
// requests) to fail.
func ForceLogout(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)

	//Validate.
	if userID <= 0 {
		output.ErrorInputInvalid("Could not determine which user's password you are changing.", w)
		return
	}

	//Update db.
	err := db.DisableLoginsForUser(r.Context(), userID)
	if err != nil {
		output.Error(err, "Could not force user to log out.", w)
		return
	}

	output.UpdateOK(w)
}
