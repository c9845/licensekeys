package users

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/users/cookieutils"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// This file handles enrolling a user in 2 Factor Authentication (TOTP) using
// a Google Authenticator type app.

// configuration options
const (
	//For identifying entry in 2FA code generator apps.
	defaultIssuer = "License Keys"

	//The maximum number of times a wrong 2FA code can be provided before we stop
	//increasing the delay between when we allow user submissions in the GUI. This
	//is done to prevent the delay from growing forever.
	max2FABadAttemps = 4
)

// easier reuse, and easier identifying of what variable is for.
var twoFATokenLength = otp.DigitsSix

// Get2FABarcode generates a QR code for enrolling a user in 2FA. This returns the QR
// code as a base64 string that will be embedded into an <img> tag using "data:" type
// in src. This only returns a QR code if user is not currently enrolled in 2FA.
func Get2FABarcode(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)

	//Validate.
	if userID < 0 {
		output.ErrorInputInvalid("Could not determine which user you are trying to manage 2 Factor Authentication for.", w)
		return
	}

	//Check if this user can manage this user's 2FA enrollment. Admins can manage any
	//users' enrollment, but non-admins can only manager their own enrollment.
	loggedInUserData, err := GetUserDataFromRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	if !loggedInUserData.Administrator && loggedInUserData.ID != userID {
		output.ErrorInputInvalid("You cannot manage the 2FA enrollment of another user.", w)
		return
	}

	//Check if 2FA is allowed and exit if 2FA is not allowed.
	as, err := db.GetAppSettings(r.Context())
	if err != nil {
		log.Println("pages.getPageConfigData", "Could not look up app settings.", err)
		return
	}
	if !as.Allow2FactorAuth {
		output.Error(db.ErrAppSettingDisabled, "2 Factor Authentication is not enabled in App Settings.", w)
		return
	}

	//Check if user is enrolled in 2FA already and exit if they are.
	cols := sqldb.Columns{
		"TwoFactorAuthEnabled",
		"Username",
		"Active",
	}
	user, err := db.GetUserByID(r.Context(), userID, cols)
	if err != nil {
		output.Error(err, "Could not look up users data.", w)
		return
	}
	if user.TwoFactorAuthEnabled {
		output.ErrorInputInvalid("This user already has 2 Factor Authentication enabled.", w)
		return
	}
	if !user.Active {
		output.ErrorInputInvalid("This user is not active.", w)
		return
	}

	//Change issuer if we are in dev mode so we don't screw up dev vs production/live
	//2FA keys since you can't have two 2FA tokens with same name in most 2FA apps.
	issuer := defaultIssuer
	if config.Data().Development {
		issuer = issuer + "_dev"
	}

	//Generate the 2FA key.
	//
	//key is the URL to be encoded in a QR code.
	//secret is the shared secret used to validate 2FA codes. It is stored in our db.
	keyOpts := totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: user.Username,
		Digits:      twoFATokenLength,
		Algorithm:   otp.AlgorithmSHA1,
		// Algorithm:   otp.AlgorithmSHA512, //Google Authenticator as of 10/2023 (I imagine since the newest major update, doesn't work with anything by SHA1)
	}
	key, err := totp.Generate(keyOpts)
	if err != nil {
		output.Error(err, "Could not generate 2FA key.", w)
		return
	}

	//Get key as an image aka QR code.
	img, err := key.Image(200, 200)
	if err != nil {
		output.Error(err, "Could not generate QR code.", w)
		return
	}

	var b bytes.Buffer
	err = png.Encode(&b, img)
	if err != nil {
		output.Error(err, "Could not get image of qr code.", w)
		return
	}

	//Get the image as bytes which is used in <img> tag src via "data:"". This way we
	//don't need to serve an img file directly; we can simply just send the image
	//data back as text that is used in src.
	imgBytes := base64.StdEncoding.EncodeToString(b.Bytes())

	//Save the secret for the user.
	err = db.Save2FASecret(r.Context(), userID, key.Secret())
	if err != nil {
		output.Error(err, "Could not save secret data for 2 Factor Authentication. Please ask an administrator to investigate the logs.", w)
		return
	}

	output.InsertOKWithData(imgBytes, w)
}

// Validate2FACode takes the 6 character 1-time code provided by a user and checks if
// it is valid given the 2FA info we have saved for the user. This is used to make
// sure that enrollment in 2FA is successful.
func Validate2FACode(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)
	token := strings.TrimSpace(r.FormValue("validationCode"))

	//Validate.
	if userID < 0 {
		output.ErrorInputInvalid("Could not determine which user you are trying to manage 2 Factor Authentication for.", w)
		return
	}
	if _, err := strconv.Atoi(token); err != nil {
		output.ErrorInputInvalid("Validation code must be numbers only.", w)
		return
	}
	if len(token) != twoFATokenLength.Length() {
		output.ErrorInputInvalid("Validation codes are exactly "+strconv.Itoa(twoFATokenLength.Length())+" characters long.", w)
		return
	}

	//Check if this user can manage this user's 2FA enrollment. Admins can manage any
	//users' enrollment, but non-admins can only manager their own enrollment.
	loggedInUserData, err := GetUserDataFromRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	if !loggedInUserData.Administrator && loggedInUserData.ID != userID {
		output.ErrorInputInvalid("You cannot manage the 2FA enrollment of another user.", w)
		return
	}

	//Look up the 2FA secret for this user.
	cols := sqldb.Columns{"TwoFactorAuthSecret"}
	user, err := db.GetUserByID(r.Context(), userID, cols)
	if err != nil {
		output.Error(err, "Could not look up users data.", w)
		return
	}

	//Validate 2FA code.
	valid := validate2FA(token, user.TwoFactorAuthSecret)
	if !valid {
		output.ErrorInputInvalid("The provided Validation Code is not valid. Please try again or refresh the page and generate a new QR code.", w)
		return
	}

	//Code is valid, set 2FA as enabled for this user.
	err = db.Enable2FA(r.Context(), userID, true)
	if err != nil {
		output.Error(err, "Could not enable 2 Factor Authentication for this user.", w)
		return
	}

	output.UpdateOK(w)
}

// validate2FA performs validation of a given 2FA token against a user's secret. This
// performs the actual checking if the token is correct.
func validate2FA(token, secret string) (valid bool) {
	valid = totp.Validate(token, secret)
	return
}

// Deactivate2FA turns 2FA off for a user.
func Deactivate2FA(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)

	//Validate.
	if userID < 0 {
		output.ErrorInputInvalid("Could not determine which user's password you are changing.", w)
		return
	}

	//Check if this user can manage this user's 2FA enrollment. Admins can manage any
	//users' enrollment, but non-admins can only manager their own enrollment.
	loggedInUserData, err := GetUserDataFromRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	if !loggedInUserData.Administrator && loggedInUserData.ID != userID {
		output.ErrorInputInvalid("You cannot manage the 2FA enrollment of another user.", w)
		return
	}

	//Turn 2FA off.
	//
	//No need to wipe the secret since it will be regenerated if 2FA is re-enabled.
	err = db.Enable2FA(r.Context(), userID, false)
	if err != nil {
		output.Error(err, "Could not deactivate 2 Factor Authentication for this user.", w)
		return
	}

	output.UpdateOK(w)
}

// browserIDCookieName is the name of the cookie that stores a browser ID used for checking
// if a 2FA token has been provided recently.
const browserIDCookieName = "browser_id"

// Set2FABrowserIDCookie saves the browser identifier to a cookie.
//
// This is used when a user is logging in to check if the user provided a 2FA token
// recently and therefore prevent users from having to provide their 2FA token overly-
// frequently.
func Set2FABrowserIDCookie(w http.ResponseWriter, browserID string, expiration time.Time) (err error) {
	cookie := http.Cookie{
		Name:     browserIDCookieName,  //
		HttpOnly: true,                 //cookie cannot be modified by client-side browser javascript.
		Secure:   false,                //this needs to be false for the demo to run since demo will most likely run on http.
		Path:     "/",                  //needed when Domain field is missing.
		SameSite: http.SameSiteLaxMode, //SameSiteStrictMode breaks browsing from history in chrome.
		Value:    browserID,            //
	}

	//Only set expiration if needed. If expiration is zero, this cookie will expire
	//at the end of the user's session (browser is closed). This is used when config
	//file is set to require 2FA token at every login.
	if !expiration.IsZero() {
		cookie.Expires = expiration
	}

	cookieutils.Set(w, cookie)
	return
}

// Get2FABrowserIDFromCookie retrieves the browser ID from a cookie.
//
// This is used whenever a user is logging in to check if the user provided their 2FA
// token recently.
func Get2FABrowserIDFromCookie(r *http.Request) (browserID string, err error) {
	browserID, err = cookieutils.Read(r, browserIDCookieName)
	return
}

// Delete2FABrowserIDCookie removes a browser ID cookie from a request/response by
// marking it as expired.
func Delete2FABrowserIDCookie(w http.ResponseWriter) {
	Set2FABrowserIDCookie(w, "", time.Now().Add(-1*time.Second))
}
