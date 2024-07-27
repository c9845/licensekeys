package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/pages"
	"github.com/c9845/licensekeys/v2/users"
	"github.com/c9845/sqldb/v3"
)

/*
This file checks if a user is already authenticated to the app (aka logged in, a
session has already been started). This makes sure the user and session are still
valid (hasn't expired, user hasn't been deactivated) and that the user's password
hasn't been changed.
*/

// Auth is used to verify a request to this app is from a logged in and active user.
// If a user's credentials are found and valid, the user is redirected to the next
// HTTP handler, otherwise an error message is returned.
//
// The Auth func should be called upon every page load or endpoint when a user is
// logged in.
//
// This does not handle external API calls! External API calls are handled via a
// separate middleware since the authentication is done differently.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Get login ID from cookie.
		//
		//Handle error in two ways so we can return a more applicable response type
		//based upon the request type. For internal app API calls (requests made
		//within the app to get data or save data), we return the same format of
		//response as every other API call. For page-view errors, we display a human
		//friendly error page.
		cv, err := users.GetLoginCookieValue(r)
		if err != nil {
			// log.Println("middleware.Auth", "could not get login cookie value", err) //diagnostics only, comment out for normal use otherwise this spams the logs when alert icon in header tries to refresh and user gets logged out by expired session.

			//Delete the login cookie so that user is forced to log in again. This
			//alleviates odd "logged in but not logged in" issues.
			users.DeleteLoginCookie(w)

			if strings.Contains(r.URL.Path, "/api/") {
				//Handle internal app API calls.
				//
				//Not using output.Error() because this spams the logs if output.Debug
				//is true for users that get logged out by an expired session but
				//alerts header icon still makes requests on an interval to try and
				//refresh icon.
				w.Header().Set("Unauthorized-Reason", "Could not identify this session and user.")
				w.WriteHeader(http.StatusUnauthorized)
				return

			} else {
				//Handle page views.
				e := pages.ErrorPage{
					PageTitle:   "Authentication Error",
					Topic:       "Could not identify this session and user.",
					Solution:    "Please try logging in again.",
					ShowLinkBtn: true,
					Link:        "/",
					LinkText:    "Log in",
				}
				pages.ShowError(w, r, e)
				return
			}
		}

		//Look up session details to see if user session is still active and not yet
		//expired. This will also get us the user's ID so we can check if user
		//themself is still active.
		ul, err := db.GetLoginByCookieValue(r.Context(), cv)
		if err != nil {
			// log.Println("middleware.Auth", "could not look up user login", err) //diagnostics only, comment out for normal use otherwise this spams the logs when alert icon in header tries to refresh and user gets logged out by expired session.

			//Delete the login cookie so that user is forced to log in again. This
			//alleviates odd "logged in but not logged in" issues.
			users.DeleteLoginCookie(w)

			if strings.Contains(r.URL.Path, "/api/") {
				//Handle internal app API calls.
				//
				//Not using output.Error() because this spams the logs if output.Debug
				//is true for users that get logged out by an expired session but
				//alerts header icon still makes requests on an interval to try and
				//refresh icon.
				w.Header().Set("Unauthorized-Reason", "Could not look up your session.")
				w.WriteHeader(http.StatusUnauthorized)
				return

			} else {
				//Handle page views.
				e := pages.ErrorPage{
					PageTitle:   "Authentication Error",
					Topic:       "Could not look up your session.",
					Solution:    "Please try logging in again.",
					ShowLinkBtn: true,
					Link:        "/",
					LinkText:    "Log in",
				}
				pages.ShowError(w, r, e)
				return
			}
		}

		//Check if user session is still active. A session can be marked inactive:
		//   1) By an admin logging a user out (via the user management page).
		//   2) If the user changed their password (or admin changed the user's
		//      password).
		//   3) If single sessions is enabled in App Settings and the user logged in
		//      on another device.
		if !ul.Active {
			//Delete the login cookie so that user is forced to log in again. This
			//alleviates odd "logged in but not logged in" issues.
			users.DeleteLoginCookie(w)

			if strings.Contains(r.URL.Path, "/api/") {
				//Handle internal app API calls.
				//
				//Not using output.Error() because this spams the logs if output.Debug
				//is true for users that get logged out by an expired session but
				//alerts header icon still makes requests on an interval to try and
				//refresh icon.
				w.Header().Set("Unauthorized-Reason", "Your session has been marked as inactive.")
				w.WriteHeader(http.StatusUnauthorized)
				return

			} else {
				//Handle page views.
				e := pages.ErrorPage{
					PageTitle:   "Authentication Error",
					Topic:       "Your session has been marked as inactive.",
					Solution:    "Please log in again.",
					ShowLinkBtn: true,
					Link:        "/",
					LinkText:    "Log in",
				}
				pages.ShowError(w, r, e)
				return
			}
		}

		// Check if session has expired because of inactivity (see config file setting
		//that sets session lifetime and code below that extends extension).
		expiration := time.Unix(ul.Expiration, 0)
		if time.Since(expiration) > 0 {
			//Delete the login cookie so that user is forced to log in again. This
			//alleviates odd "logged in but not logged in" issues.
			users.DeleteLoginCookie(w)

			if strings.Contains(r.URL.Path, "/api/") {
				//Handle internal app API calls.
				//
				//Not using output.Error() because this spams the logs if output.Debug
				//is true for users that get logged out by an expired session but
				//alerts header icon still makes requests on an interval to try and
				//refresh icon.
				w.Header().Set("Unauthorized-Reason", "Your session has expired.")
				w.WriteHeader(http.StatusUnauthorized)
				return

			} else {
				//Handle page views.
				e := pages.ErrorPage{
					PageTitle:   "Authentication Error",
					Topic:       "Your session has expired.",
					Solution:    "Please log in again.",
					ShowLinkBtn: true,
					Link:        "/",
					LinkText:    "Log in",
				}
				pages.ShowError(w, r, e)
				return
			}
		}

		//Look up user data and make sure user is still active. An admin could have
		//marked a user as inactive while a session was still active.
		cols := sqldb.Columns{db.TableUsers + ".Active"}
		u, err := db.GetUserByID(r.Context(), ul.UserID, cols)
		if err != nil {
			// log.Println("middleware.Auth", "could not look up user", err) //diagnostics only, comment out for normal use otherwise this spams the logs when alert icon in header tries to refresh and user gets logged out by expired session.

			//Delete the login cookie so that user is forced to log in again. This
			//alleviates odd "logged in but not logged in" issues.
			users.DeleteLoginCookie(w)

			if strings.Contains(r.URL.Path, "/api/") {
				//Handle internal app API calls.
				//
				//Not using output.Error() because this spams the logs if output.Debug
				//is true for users that get logged out by an expired session but
				//alerts header icon still makes requests on an interval to try and
				//refresh icon.
				w.Header().Set("Unauthorized-Reason", "Could not determine if your user account is still active.")
				w.WriteHeader(http.StatusUnauthorized)
				return

			} else {
				//Handle page views.
				e := pages.ErrorPage{
					PageTitle:   "Authentication Error",
					Topic:       "Could not determine if your user account is still active.",
					Solution:    "Please contact an administrator.",
					ShowLinkBtn: false,
				}
				pages.ShowError(w, r, e)
				return
			}
		}

		if !u.Active {
			//Delete the login cookie so that user is forced to log in again. This
			//alleviates odd "logged in but not logged in" issues.
			users.DeleteLoginCookie(w)

			if strings.Contains(r.URL.Path, "/api/") {
				//Handle internal app API calls.
				//
				//Not using output.Error() because this spams the logs if output.Debug
				//is true for users that get logged out by an expired session but
				//alerts header icon still makes requests on an interval to try and
				//refresh icon.
				w.Header().Set("Unauthorized-Reason", "Your user account has been marked as inactive.")
				w.WriteHeader(http.StatusUnauthorized)
				return

			} else {
				//Handle page views.
				e := pages.ErrorPage{
					PageTitle:   "Authentication Error",
					Topic:       "Your user account has been marked as inactive.",
					Solution:    "Please contact an administrator.",
					ShowLinkBtn: false,
				}
				pages.ShowError(w, r, e)
				return
			}
		}

		//User and login/session has been validated. Extend the session expiration to
		//keep user logged in (expiration time resets each time user visits a new page
		//in the app). Extending the expiration updates the database, our source of
		//truth, and cookie.
		//
		//User sessions aren't extended for calls to the internal app API endpoints
		//because we only want to extend on page views. For example, if we had an
		//API endpoint that was hit every 60 seconds, then a user's session would
		//never expire.
		if !strings.Contains(r.URL.Path, "/api/") {
			newExpiration := time.Now().Add(time.Duration(config.Data().LoginLifetimeHours) * time.Hour)
			err = ul.ExtendLoginExpiration(r.Context(), newExpiration.Unix())
			if err != nil {
				log.Println("middleware.Auth", "could not extend db expiration", err)

				//Not returning on error here since the user can still use app, their
				//session will just expire sooner than expected.
			}

			users.SetLoginCookieValue(w, ul.CookieValue, newExpiration)
		}

		//Save user ID to context for use in activity logging. This just reduces
		//the workload in LogActivity2() since we can just check if this key exists
		//meaning the request was made via an API key.
		ctx := context.WithValue(r.Context(), UserIDCtxKey, ul.UserID)
		r = r.WithContext(ctx)

		//Move to next middleware or handler.
		next.ServeHTTP(w, r)
	})
}

type userIDCtxKeyType string

// UserIDCtxKey is used to identify a User ID in the request context.
const UserIDCtxKey userIDCtxKeyType = "user-id"
