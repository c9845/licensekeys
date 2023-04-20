/*
Package middleware handles authentication, user permissions, and any other tasks
that occur with a request to this app.

This file defines functions for checking user permissions. A function must be
created for each permission to be checked (see users table schema).

Try to keep the funcs in the same order as the permissions in the gui and as defined in
the user struct. Just easier to lookup and find.
*/
package middleware

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/pages"
	"github.com/c9845/licensekeys/v2/users"
	"github.com/c9845/output"
)

// errPermissionRefused is returned when a user does not have the correct permission
// to view a page or endpoint or perform an action.
var errPermissionRefused = errors.New("middleware: user does not have permission to this page or to perfom this action")

// refuseAccess sends back the correct form of permission denied error message based
// upon what kind of request this is. We have to handle two types of requests: api/ajax
// requests or gui page requests.
func refuseAccess(w http.ResponseWriter, r *http.Request, permission string, u db.User) {
	msg := "You do not have the '" + permission + "' permission. Please contact an administrator."

	//Determine if this request is for a page or api endpoint and send back error
	//formatted correctly for request type.
	if strings.Contains(r.URL.Path, "api") {
		output.Error(errPermissionRefused, msg, w)
		return
	}

	pd := pages.PageData{
		UserData: u,
		Data:     msg,
	}
	pages.Show(w, "app", "permission-error", pd)
}

// verifyAccessError returns an error when an error occurs when trying to verify a
// user's permissions.
//
// User data is not used since this func can be used after an error occured while
// trying to look up a user and therefore we wouldn't have user data to use in this
// func.
func verifyAccessError(w http.ResponseWriter, r *http.Request, permission string, err error) {
	log.Println("middleware.verifyAccessError", r.URL.Path, permission, err)

	ep := pages.ErrorPage{
		PageTitle: "Could Not Verify Permission",
		Topic:     "An error occured while trying to verify you have the '" + permission + "' permission.",
		Solution:  "Please contact an administrator to investigate this error.",
	}

	//Determine if this request is for a page or api endpoint and send back error
	//formatted correctly for request type.
	if strings.Contains(r.URL.Path, "api") {
		output.Error(err, ep.Topic, w)
		return
	}

	pages.ShowError(w, r, ep)
}

// Administrator checks if the user has this permission.
func Administrator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const p = "Administrator"

		//Get user data.
		u, err := users.GetUserDataByRequest(r)
		if err != nil {
			verifyAccessError(w, r, p, err)
			return
		}

		//Check if user has required permission.
		if !u.Administrator {
			refuseAccess(w, r, p, u)
			return
		}

		//User has permission.
		next.ServeHTTP(w, r)
	})
}

// CreateLicenses check if the user has this permission.
func CreateLicenses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const p = "CreateLicenses"

		u, err := users.GetUserDataByRequest(r)
		if err != nil {
			verifyAccessError(w, r, p, err)
			return
		}

		if !u.CreateLicenses {
			refuseAccess(w, r, p, u)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ViewLicenses check if the user has this permission.
func ViewLicenses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const p = "ViewLicenses"

		u, err := users.GetUserDataByRequest(r)
		if err != nil {
			verifyAccessError(w, r, p, err)
			return
		}

		if !u.ViewLicenses {
			refuseAccess(w, r, p, u)
			return
		}

		next.ServeHTTP(w, r)
	})
}
