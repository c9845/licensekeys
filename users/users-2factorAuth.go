/*
Package users handles interacting with users of the app.

This file handles enrolling a user in 2 Factor Authentication (TOTP) using
a Google Authenticator type app.
*/
package users

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/utils"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v2"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// configuration options
const (
	//secretLength is the length of the shared secret
	//20 is 160 bytes which is the recommended length per RFC4226
	secretLength = 20

	//default stuff for uri
	defaultIssuer = "License Keys"

	//the length of the one time codes provided for 2 factor auth
	twoFATokenLength = 6

	//the maximum number we use to build the delay for bad 2fa attempts
	max2FABadAttemps = 4

	//this is the name of the cookie saved to identify a browser so that we can
	//check if user already provided 2fa on this browser recently.  makes it so
	//user doesn't have to provide 2fa token upon every login to same browser
	twoFACookieName = "2fa_browser_id"
)

// Get2FABarcode generates a QR code for enrolling a user in 2FA. This returns the QR
// code as a base64 string that will be embedded into an <img> tag using data: type
// in src. This only returns a QR code if user is not currently enrolled in 2FA.
func Get2FABarcode(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)

	//Validate.
	if userID < 0 {
		output.ErrorInputInvalid("Could not determine which user's password you are changing.", w)
		return
	}

	//Check if 2fa is allowed and exit if 2fa is not allowed.
	as, err := db.GetAppSettings(r.Context())
	if err != nil {
		log.Println("pages.getPageConfigData", "Could not look up app settings.", err)
		return
	}
	if !as.Allow2FactorAuth {
		output.Error(db.ErrAppSettingDisabled, "2 Factor Authentication is not enabled in App Settings.", w)
		return
	}

	//Check if user is enrolled in 2fa already and exit if they are.
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
	//2fa keys since you can't have two 2fa tokens with same name in most 2fa apps.
	issuer := defaultIssuer
	if config.Data().Development {
		issuer = issuer + "_dev"
	}

	//Generate the 2fa key.
	//key is the url to be encoded in a qr code.
	//secret is generated automatically and retrieved from the key for storing in db.
	keyOpts := totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: user.Username,
		Period:      30,
		SecretSize:  secretLength,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA512,
	}
	key, err := totp.Generate(keyOpts)
	if err != nil {
		output.Error(err, "Could not generate 2fa key.", w)
		return
	}

	//Get key as an image aka QR code.
	img, err := key.Image(200, 200)
	if err != nil {
		output.Error(err, "Could not generate qr code.", w)
		return
	}

	var b bytes.Buffer
	err = png.Encode(&b, img)
	if err != nil {
		output.Error(err, "Could not get image of qr code.", w)
		return
	}

	//Get the image as bytes which is used in img tag src via data:. This way we don't
	//need to serve an img file directly; we can simply just send the image data back
	//as text that is used in src.
	imgBytes := base64.StdEncoding.EncodeToString(b.Bytes())

	//Save the secret for the user.
	err = db.Save2FASecret(r.Context(), userID, key.Secret())
	if err != nil {
		output.Error(err, "Could not save secret data for 2 Factor Authentication.  Investigate the logs.", w)
		return
	}

	output.InsertOKWithData(imgBytes, w)
}

// Validate2FACode takes the 6 character 1-time code provided by a user and checks if
// it is valid given the 2fa info we have saved for the user. This is used to make sure
// that enrollment in 2fa is successful.
func Validate2FACode(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	userID, _ := strconv.ParseInt(r.FormValue("userID"), 10, 64)
	token := strings.TrimSpace(r.FormValue("validationCode"))

	//Validate.
	if userID < 0 {
		output.ErrorInputInvalid("Could not determine which user's password you are changing.", w)
		return
	}
	if _, err := strconv.Atoi(token); err != nil {
		output.ErrorInputInvalid("Validation code must be numbers only.", w)
		return
	}
	if len(token) != twoFATokenLength {
		output.ErrorInputInvalid("Validation codes are exactly "+strconv.Itoa(twoFATokenLength)+" characters long.", w)
		return
	}

	//Get 2fa secret for this user.
	cols := sqldb.Columns{"TwoFactorAuthSecret"}
	user, err := db.GetUserByID(r.Context(), userID, cols)
	if err != nil {
		output.Error(err, "Could not look up users data.", w)
		return
	}

	//Validate 2fa code.
	valid := validate2FA(token, user.TwoFactorAuthSecret)
	if !valid {
		output.ErrorInputInvalid("The provided Validation Code is not valid.  Please try again or refresh the page and generate a new QR code.", w)
		return
	}

	//Code is valid, set 2fa as enabled for this user.
	err = db.Enable2FA(r.Context(), userID, true)
	if err != nil {
		output.Error(err, "Could not enable 2 Factor Authentication for this user.", w)
		return
	}

	output.UpdateOK(w)
}

// validate2FA performs validation of a given 2fa token and a user's secret. This
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

	//Turn 2fa off.
	//No need to wipe the secret since it will be regenerated anyway if 2fa is re-enabled.
	err := db.Enable2FA(r.Context(), userID, false)
	if err != nil {
		output.Error(err, "Could not deactivate 2 Factor Authentication for this user.", w)
		return
	}

	output.UpdateOK(w)
}

// save2FABrowserIDCookie saves the cookie used to identify a browser. This is used to
// help identify a browser to see if a 2FA token was recently used on this browser
// already and therefore not require user to provide it again.
//
// This is kept separate from normal login ID cookie since some users may not have 2FA
// enabled. Plus, this allows for 2FA cookie to expire on a different time frequency
// than the login ID cookie.
//
// Everything in ab will be provided except the cookie value which is generated here.
func save2FABrowserIDCookie(ctx context.Context, w http.ResponseWriter, ab db.AuthorizedBrowser) (err error) {
	//Get random value to store in cookie to identify this browser/session.
	cv, err := utils.RandString(32)
	if err != nil {
		log.Println("users.save2FABrowserIDCookie", "could not get cookie value for 2fa browser id, defaulting to timetamp", err)
		cv = strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	//Add userID to random value since we do this for login cookie. This provided a bit
	//of additional collision resistance since the chance of one user getting the exact
	//same random value generated is higher, although very slightly, then multiple users
	//getting the same random value. This is probably not necessary. We aren't using a
	//salt here since we aren't hashing the value before its use and therefore the salt
	//would just make the cookie value longer without adding any randomness since it
	//would be the same between cookies.
	cv = strconv.FormatInt(ab.UserID, 10) + "_" + cv

	//Create the cookie.
	cookie := http.Cookie{
		Name:     twoFACookieName,
		HttpOnly: true,
		Secure:   false, //this needs to be false for the demo to run since demo will most likely run on http.
		Domain:   config.Data().FQDN,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		Value:    cv,
	}

	//Only set expiration if it is at least 1 day. If lifetime is 0, that means that
	//user should provide token each time the log into the app.
	if config.Data().TwoFactorAuthLifetimeDays > 0 {
		cookie.Expires = time.Now().AddDate(0, 0, config.Data().TwoFactorAuthLifetimeDays)
	}

	//Set the cookie.
	http.SetCookie(w, &cookie)

	//Save the authorized browser to the db.
	ab.Cookie = cv
	err = ab.Insert(ctx)
	return
}

func delete2FABrowserCookie(w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:     twoFACookieName,
		HttpOnly: true,
		Secure:   false, //this needs to be false for the demo to run.  see session.go for more info.
		Domain:   config.Data().FQDN,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		Value:    "",
		Expires:  time.Now(),
		MaxAge:   -1,
	}

	http.SetCookie(w, &cookie)
}
