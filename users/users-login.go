package users

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/users/cookieutils"
	"github.com/c9845/licensekeys/v2/users/pwds"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
)

// badPasswordAttemptsMax helps determine the maximum delay a user will experience if
// they provide a bad password over and over. This value is multiplied against a time
// value, i.e.: 1 second, to determine the delay between providing a bad password and
// when the user can try again. Setting an upper limit prevents the delay from growing
// endlessly.
//
// Increase this number to make brute forcing passwords more expensive.
const badPasswordAttemptsMax = 5

// Login handles authentication a user logging in to the app. This handles password
// login and 2fa login.
func Login(w http.ResponseWriter, r *http.Request) {
	//Get inputs.
	username := r.FormValue("username")
	password := r.FormValue("password")
	twoFAToken := strings.TrimSpace(r.FormValue("twoFAToken"))

	//Validation.
	if len(username) < 5 {
		output.ErrorInputInvalid("You must provide an email address as a username.", w)
		return
	}
	if len(password) < config.Data().MinPasswordLength {
		output.ErrorInputInvalid("The credentials you provided are invalid.", w)
		return
	}

	//Look up user's data via username.
	//
	//We will use this delay login if user has provided incorrect password a few
	//times, check if user account is active, and check if password provided is valid.
	u, err := db.GetUserByUsername(r.Context(), username, sqldb.Columns{"*"})
	if err != nil {
		output.Error(err, "Could not determine if a user with your username exists.", w)
		return
	}

	//Delay login based on if previous password attempts were bad.
	if u.BadPasswordAttempts > 0 {
		delay := time.Second * time.Duration(u.BadPasswordAttempts)
		log.Println("users.Login", "delaying password auth for:", delay, username)
		time.Sleep(delay)
	}

	//Check if user is active.
	if !u.Active {
		output.ErrorInputInvalid("Your account is inactive. Please contact your administrator.", w)
		return
	}

	//Check password provided.
	//
	//If password is bad, increment login delay counter up to max. If max delay is
	//reached, just continue to use it. Using a max delay is important so delay doesn't
	//just grow endlessly.
	ok, err := pwds.IsValid(password, u.Password)
	if err != nil && err != pwds.ErrBadPassword {
		output.Error(err, "There was an issue verifying the provided password. Please ask an administrator to investigate in the app's logs.", w)
		return
	}
	if !ok {
		if u.BadPasswordAttempts < badPasswordAttemptsMax {
			newBadAttempts := u.BadPasswordAttempts + 1
			err := db.SetPasswordBadAttempts(r.Context(), u.ID, newBadAttempts)
			if err != nil {
				log.Println("users.Login", "could not increment password bad attempts", err)
				//not returning since this isn't an end of the world situation
			}
		}

		output.Error(err, "Could not verify your username and password. Please make sure both are correct.", w)
		return
	}

	//
	//At this point, we know that the user is active and that the provided username
	//and password match and are correct. Now, we have to handle 2 factor auth given
	//it is enabled.
	//

	//Get app setttings to for checking 2 factor auth stuff.
	as, err := db.GetAppSettings(r.Context())
	if err != nil {
		output.Error(err, "Could not verify if 2 Factor Authentication is enabled.", w)
		return
	}

	//Get data from request to identify user in more detail. This is used for
	//"authorized browsers" feature of 2 Factor Auth and for saving to user logins
	//history for diagnostics/auditing.
	ip := getIPFormatted(r)
	ua := r.UserAgent()

	//Define custom response message types. These message types are used client side
	//in login.ts for handling what should happen next in the GUI.
	const (
		msgType2FAForced        = "login2FAForced"
		msgType2FATokenRequired = "login2FATokenRequired"
		msgTypeLoginOK          = "loginOK"
	)

	//Check if 2FA is forced upon users and this user doesn't have 2FA enabled. In
	//this case, user has to see an administrator first to enable 2FA before being
	//able to log in.
	if as.Allow2FactorAuth && as.Force2FactorAuth && !u.TwoFactorAuthEnabled {
		log.Println("users.Login", "2FA is required but not enabled for user")
		output.Success(msgType2FAForced, "Two Factor Authentication (2FA) is required but not enabled for your user account. Please see an administrator to enable 2FA.", w)
		return
	}

	//Check if 2FA is enabled for app and user. If so, we need to check if the user's
	//browser is already remembered/authenticated (a 2FA token was provided recently).
	//If not, we need to request the 2FA token.
	if as.Allow2FactorAuth && u.TwoFactorAuthEnabled {
		//Handle when 2FA token isn't provided. Either user is logging in to app in
		//this browser for the first time, in which case we will request 2FA from user,
		//or browser is rememebered by a previously provided 2FA, in which case we
		//will need to validate remembered browser.
		if twoFAToken == "" {
			//User did not provide 2FA token.

			//Look up cookie to see if this browser is remembered. If cookie cannot be
			//found, request 2FA token.
			browserID, err := Get2FABrowserIDFromCookie(r)
			if err != nil {
				log.Println("users.Login", "could not find browser id cookie, requiring 2fa")
				output.Success(msgType2FATokenRequired, nil, w)
				return
			}

			//Cookie found, verify it. If cookie cannot be verified, request 2FA token.
			authedBrowser, err := db.GetAuthorizedBrowser(r.Context(), u.ID, ip, browserID, true)
			if err == sql.ErrNoRows {
				log.Println("users.Login", "could not find browser for given cookie, this is odd and should never happen", err)
				output.Success(msgType2FATokenRequired, nil, w)
				return
			} else if err != nil {
				log.Println("users.Login", "could not verify if browser authorized, requiring 2fa", err)
				output.Success(msgType2FATokenRequired, nil, w)
				return
			}

			//Check if browser was authed within time limit. We only authenticate a
			//browser for a set amount of time to force users to provide 2FA token
			//every so often for improved security. If browser has not been
			//authenticated recently, request 2FA token.
			authedBrowserAge := time.Since(time.Unix(authedBrowser.Timestamp, 0))
			maxBrowserAge := time.Duration(config.Data().TwoFactorAuthLifetimeDays * 24 * int(time.Hour))
			if authedBrowserAge > maxBrowserAge {
				log.Println("users.Login", "removing expired browser id cookie")
				Delete2FABrowserIDCookie(w)
				output.Success(msgType2FATokenRequired, nil, w)
				return
			}

			//
			//At this point, we know that the user is active, that the provided
			//username and password match and are correct, and 2FA token is not needed
			//because user has provided the token recently for this browser. Now we
			//just need to record the login/session and redirect the user to the main
			//logged in page.
			//

		} else {
			//User provided 2FA token.

			//Delay as needed. This helps prevent brute force attempts of the 2FA token.
			if u.TwoFactorAuthBadAttempts > 0 {
				delay := time.Second * time.Duration(2*u.TwoFactorAuthBadAttempts)
				log.Println("users.Login", "delaying 2FA auth for:", delay)
				time.Sleep(delay)
			}

			//Validate the 2FA token.
			//First, we do some simple checks for format since we know the 2FA token
			//is 6 numbers. Then, we verify the token itself. If token is not valid,
			//return an error telling user to try again.
			if len(twoFAToken) != twoFATokenLength.Length() {
				output.ErrorInputInvalid("The 2 Factor Authentication code you provided is not the correct length. It must be exactly "+strconv.Itoa(twoFATokenLength.Length())+" numbers long.", w)
				return
			}
			if _, err := strconv.Atoi(twoFAToken); err != nil {
				output.Error(err, "The 2 Factor Authentication code is not valid. It must be numbers only.", w)
				return
			}
			if valid := validate2FA(twoFAToken, u.TwoFactorAuthSecret); !valid {
				if u.TwoFactorAuthBadAttempts < max2FABadAttemps {
					newBadAttempts := u.TwoFactorAuthBadAttempts + 1
					err := db.Set2FABadAttempts(r.Context(), u.ID, newBadAttempts)
					if err != nil {
						log.Println("users.Login", "could not increment 2fa bad attempts", err)
						//not returning since this isn't an end of the world situation
					}
				}

				output.ErrorInputInvalid("The 2 Factor Authentication code you provided is invalid.  Please try again.", w)
				return
			}

			//
			//At this point we know the 2FA token is valid. We need to save the
			//authorized browser for referencing in the future to reduce the number
			//of times a user has to provide a 2FA token.
			//

			//Get a unique, random identifier for this browser. This is used in
			//place of a numeric ID (ex: from authorized_browsers table) to make it
			//harder to guess.
			const length = 32
			b := make([]byte, length)
			_, err = rand.Read(b)
			if err != nil {
				log.Println("users.Login", "could not generate browser ID", err)
				//not returned on error since this isn't an end of the world situation,
				//user will just need to provide 2FA token upon next login.
			}
			browserID := base64.StdEncoding.EncodeToString(b)

			//Set the cookie that identifies this browser.
			//
			//This cookie will be used to prevent a user from having to provide their
			//2FA token upon every login to reduce user friction when logging in.
			//
			//This cookie has a set expiration, based on config file setting. The
			//expiration does not get extended each time a user logs in.
			expiration := time.Time{}
			if config.Data().TwoFactorAuthLifetimeDays > 0 {
				expiration = time.Now().AddDate(0, 0, config.Data().TwoFactorAuthLifetimeDays)
			}
			Set2FABrowserIDCookie(w, browserID, expiration)

			//Record that 2FA was provided to our database.
			ab := db.AuthorizedBrowser{
				UserID:    u.ID,
				RemoteIP:  ip,
				UserAgent: ua,
				Timestamp: time.Now().Unix(),
				Cookie:    browserID,
			}
			err = ab.Insert(r.Context())
			if err != nil {
				log.Println("users.Login", "could not save browser ID", err)
				//not returning since this isn't an end of the world situation, user
				//will just have to provid e 2FA token upon next login.
			}

			//Reset bad 2FA token counter.
			err := db.Set2FABadAttempts(r.Context(), u.ID, 0)
			if err != nil {
				log.Println("users.Login", "could not reset 2fa bad attempts", err)
				//not returning since this isn't an end of the world situation
			}

			//
			//At this point, we know that the user is active, that the provided
			//username and password match and are correct, and 2FA token was provided
			//and was correct. Now we just need to record the login/session and redirect
			//the user to the main logged in page.
			//

		} //end if: did user provide 2FA token
	} //end if: 2FA is enabled for app and user.

	//
	//The only other 2FA option that hasn't been handled at this point is if 2FA is
	//disabled for the app. In this case, we don't have to do anything else since user
	//& password have been validated. Now, we just need to records the login/session
	//and redirect the user to the main logged in page.
	//

	//Handle single sessions. If single session is enabled, mark all other sessions
	//as inactive so they cannot be used. This is a security feature to prevent users
	//from being logged into the app in more than one place at a time. This is done
	//BEFORE saving the current login/session so we don't disable it by mistake.
	if as.ForceSingleSession {
		err := db.DisableLoginsForUser(r.Context(), u.ID)
		if err != nil {
			log.Println("users.Login", "could not disable all active/non-expired sessions b/c of forcing single session", err)
			//this isn't a huge error so we aren't returning
		}
	}

	//Get unique, random identifier for this user session. This is used in place of
	//a numeric ID (ex: ID from user_logins table) to make it harder to guess.
	const length = 32
	b := make([]byte, length)
	_, err = rand.Read(b)
	if err != nil {
		output.Error(err, "Could not initialize user session.", w)
		return
	}
	sessionID := base64.StdEncoding.EncodeToString(b)

	//Set the cookie that identifies this session.
	//
	//This cookie is used to identify the user's session and keeps the user logged in
	//to the app. From this cookie we can get the user's username, permissions, and
	//any other details.
	//
	//While we set an expiration on this cookie, and the value matches the expiration
	//we saved to the database, we only rely on the database value for validity since
	//it cannot be modified client side. The expiration (cookie and database) are
	//updated as the user uses the app to keep the user logged in while they are
	//active.
	expiration := time.Now().Add(time.Duration(config.Data().LoginLifetimeHours) * time.Hour)
	SetUserSessionIDCookie(w, sessionID, expiration)

	//Record the session to our database.
	twoFATokenProvided := as.Allow2FactorAuth && u.TwoFactorAuthEnabled && twoFAToken != ""

	ul := db.UserLogin{
		UserID:             u.ID,
		RemoteIP:           ip,
		UserAgent:          ua,
		TwoFATokenProvided: twoFATokenProvided,
		CookieValue:        sessionID,
		Active:             true,
		Expiration:         expiration.Unix(),
	}
	err = ul.Insert(r.Context())
	if err != nil {
		output.Error(err, "Could not save login and therefore could not log user into app. Please see an administrator for help.", w)
		return
	}

	//Reset bad password counter since user has successfully logged in. We don't want
	//to user to experience a delayed login time the next login.
	err = db.SetPasswordBadAttempts(r.Context(), u.ID, 0)
	if err != nil {
		log.Println("users.Login", "could not reset password bad attempts", err)
		//not returning since this isn't an end of the world situation
	}

	//Respond successfully to request. This will cause the JS code that made this
	//request to redirect the user to the main logged in page.
	output.Success(msgTypeLoginOK, nil, w)
}

// getIPFormatted formats the IP in an http request. This is used in Login to make sure
// we always get the same format for the IP to check if user has already authorized
// browser via 2fa. Note that user's real IP is probably being put in header since
// there should be a proxy in front of this app.
func getIPFormatted(r *http.Request) (ip string) {
	ip = r.RemoteAddr

	if v, ok := r.Header["X-Forwarded-For"]; ok {
		ip = strings.Join(v, "")
	}

	if strings.Contains(ip, "::") {
		//development on local desktop
		ip = "localhost"
	} else if strings.Contains(ip, "[") && strings.Contains(ip, "]") {
		//development, ipv6
		idxOpenBracket := strings.Index(ip, "[")
		idxCloseBracket := strings.Index(ip, "]")
		ip = ip[idxOpenBracket:idxCloseBracket]
	} else if strings.Contains(ip, ":") {
		//remote ipv4 server
		idx := strings.LastIndex(ip, ":")
		ip = r.RemoteAddr[:idx]
	}

	return
}

// sessionIDCookieName is the name of the cookie used to store the session ID.
const sessionIDCookieName = "session_id"

// SetUserSessionIDCookie saves the user session identifier to a cookie.
//
// This is used when a user is logging in, when extending a user's session (see
// middleware, and indirectly when logging a user out by marking the cookie as
// expired).
//
// This cookie identifies a user's session and is used to keep the user logged in. The
// identifier in this cookie references the user session saved to our database. The
// expiration should match the value saved to the database.
func SetUserSessionIDCookie(w http.ResponseWriter, sessionID string, expiration time.Time) (err error) {
	cookie := http.Cookie{
		Name:     sessionIDCookieName,  //
		HttpOnly: true,                 //cookie cannot be modified by client-side browser javascript.
		Secure:   false,                //this needs to be false for the demo to run since demo will most likely run on http.
		Path:     "/",                  //needed when Domain field is missing.
		SameSite: http.SameSiteLaxMode, //SameSiteStrictMode breaks browsing from history in chrome.
		Value:    sessionID,            //
		Expires:  expiration,           //
	}
	cookieutils.Set(w, cookie)
	return
}

// GetUserSessionIDFromCookie retrieves the user session ID from a cookie.
//
// This is used whenever user authorization in the app is needed, specifically in
// middleware to validate that a user session is currently active.
func GetUserSessionIDFromCookie(r *http.Request) (sessionID string, err error) {
	sessionID, err = cookieutils.Read(r, sessionIDCookieName)
	return
}

// DeleteSessionIDCookie removes a session ID cookie from a request/response by
// marking it as expired.
func DeleteSessionIDCookie(w http.ResponseWriter) {
	SetUserSessionIDCookie(w, "", time.Now().Add(-1*time.Second))
}
