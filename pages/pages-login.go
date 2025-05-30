package pages

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/c9845/licensekeys/v3/db"
	"github.com/c9845/licensekeys/v3/users"
)

//This file specifically handles the login page. This functionality was broken out
//into a separate file since it is a bit complex (to handle auto-login for non-expired
//sessions, 2 Factor Auth).

// Login shows the login page to the app. This also checks if the user is already logged
// in and redirects the user to the main logged in page if so.
func Login(w http.ResponseWriter, r *http.Request) {
	//Look up login cookie. If it exists, check if the login is valid and active and
	//if so, redirect user.
	cv, err := users.GetUserSessionIDFromCookie(r)
	if err == nil && cv != "" {
		ul, err := db.GetLoginByCookieValue(r.Context(), cv)
		if err == sql.ErrNoRows {
			users.DeleteSessionIDCookie(w)

			http.Redirect(w, r, "/?ref=loginNotFound", http.StatusFound)
			return

		} else if err != nil {
			users.DeleteSessionIDCookie(w)
			log.Println("pages.Login", "could not get login by cookie value", err)

			errPage := ErrorPage{
				PageTitle: "Login Error",
				Topic:     "Could not determine if you are already logged in. Please refresh this page.",
				Solution:  "Please ask an administrator for help.",
			}
			ShowError(w, r, errPage)
			return
		}

		if !ul.Active {
			users.DeleteSessionIDCookie(w)

			http.Redirect(w, r, "/?ref=sessionNotActive", http.StatusFound)
			return
		}

		expiration := time.Unix(ul.Expiration, 0)
		if time.Since(expiration) > 0 {
			users.DeleteSessionIDCookie(w)

			http.Redirect(w, r, "/?ref=sessionExpired", http.StatusFound)
			return
		}

		http.Redirect(w, r, "/app/?ref=autoLogIn", http.StatusFound)
		return
	}

	//Show user the login page.
	Show(w, "app/login.html", nil)
}
