package middleware

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/apikeys"
	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/pages"
	"github.com/c9845/licensekeys/v2/users"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v3"
)

// This file checks if a user is already authenticated to the app (has a session set),
// that the user and session are valid, that the user is active, and that the user's
// password hasn't changed.
//
// The Auth func should be called upon every page load or endpoint when a user is logged
// in.

var errLoginNotValid = errors.New("login inactive or expired")

// Auth is used to verify a request to this app is authenticated via a user profile.
// API authentications are handled via middleware.ExternalAPI since api reachable
// endpoints are hosted on /api/v1/. If a user is found then the next middleware or
// handler is performed, otherwise a user is redirected to a login page.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Check if user provided an api key and tell them they are trying to reach a
		//non-public api accessible endpoint.
		key := strings.ToUpper(strings.TrimSpace(r.FormValue("apiKey")))
		if len(key) != 0 {
			log.Println("bad api endpoint:", r.URL.Path)
			output.Error(apikeys.ErrNonPublicEndpoint, "You are trying to access a non-public endpoint via the API.", w)
			return
		}

		//Get login ID from cookie.
		//Handling error in two ways so we can handle internal app api calls and actual
		//page loads that a user would see. Either way, we try deleting the login ID
		//cookie so user can try logging in again.
		cv, err := users.GetLoginCookieValue(r)
		if err != nil {
			users.DeleteLoginCookie(w)

			if strings.Contains(r.URL.Path, "/api/") {
				//handle api calls.
				output.Error(err, "Could not identify this session and user. Please try logging in again.", w)
			} else {
				//handle pages.
				e := pages.ErrorPage{
					PageTitle:   "Authentication Error",
					Topic:       "Please try logging in again",
					Solution:    "",
					ShowLinkBtn: true,
					Link:        "/",
					LinkText:    "Log in",
				}
				pages.ShowError(w, r, e)
			}

			return
		}

		//Look up session details to see if it is active and not expired.
		//Session could be made inactive if:
		// 1) app settings force single session and user logged in elsewhere,
		// 2) user was forced to log out by admin,
		// 3) user password was changed.
		// 4) Session expires because of inactivity (see config file setting that sets
		//     lifetime and extension code below).
		ul, err := db.GetLoginByCookieValue(r.Context(), cv)
		if err != nil {
			users.DeleteLoginCookie(w)

			if strings.Contains(r.URL.Path, "/api/") {
				//handle api calls.
				output.Error(err, "Could not verify your session is active. Please try logging in again.", w)
			} else {
				//handle pages.
				e := pages.ErrorPage{
					PageTitle:   "Authentication Error",
					Topic:       "Could not verify your session is active.",
					Solution:    "Please try logging in again.",
					ShowLinkBtn: true,
					Link:        "/",
					LinkText:    "Log in",
				}
				pages.ShowError(w, r, e)
			}

			return
		}
		if !ul.Active {
			users.DeleteLoginCookie(w)

			if strings.Contains(r.URL.Path, "/api/") {
				output.Error(errLoginNotValid, "Your login has expired. Please log in again.", w)
			} else {
				http.Redirect(w, r, "/?ref=loginNotActive", http.StatusFound)
			}
			return
		}

		expiration := time.Unix(ul.Expiration, 0)
		if time.Since(expiration) > 0 {
			users.DeleteLoginCookie(w)

			if strings.Contains(r.URL.Path, "/api/") {
				output.Error(errLoginNotValid, "Your login has expired. Please log in again.", w)
			} else {
				http.Redirect(w, r, "/?ref=loginExpired", http.StatusFound)
			}

			return
		}

		//Look up user data and make sure user is still active.
		//User can be marked inactive by administrator.
		cols := sqldb.Columns{db.TableUsers + ".Active"}
		u, err := db.GetUserByID(r.Context(), ul.UserID, cols)
		if err != nil {
			users.DeleteLoginCookie(w)

			output.Error(err, "Could not verify your user active is active. Please try logging in again.", w)
			return
		}
		if !u.Active {
			users.DeleteLoginCookie(w)

			pd := pages.PageData{
				UserData: u,
				Data:     "Your user account is not active.  Please contact an administrator.",
			}
			pages.Show(w, "app", "error", pd)
			return
		}

		//User and login/session has been validated. Extend the session expiration to
		//keep user logged in (expiration time resets each time user visits a new page
		//in the app). Extending the expiration updates the database, our source of
		//truth, and cookie.
		if !strings.Contains(r.URL.Path, "/api/") {
			newExpiration := time.Now().Add(time.Duration(config.Data().LoginLifetimeHours) * time.Hour)
			err = ul.ExtendLoginExpiration(r.Context(), newExpiration.Unix())
			if err != nil {
				log.Println("middleware.Auth", "could not extend db expiration", err)
				//not returning error here since user can still use app, their login will just expire sooner than expected
			}

			users.SetLoginCookieValue(w, ul.CookieValue, newExpiration)
		}

		//move to next middleware or handler
		next.ServeHTTP(w, r)
	})
}
