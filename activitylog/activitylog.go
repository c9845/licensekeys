/*
Package activitylog handles the logging of user actions performed within the app for
diagnostic and auditing purposes. The activity log records each page view and each
api endpoint hit. The request data is saved but the response data is not (because we
would need to write our own http response writer interface). Activity logging also
works for public api calls.
*/
package activitylog

import (
	"net/http"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/output"
)

// Clear handles deleting rows from the activity log table. This is usually only done
// from the admin tools page to reduce the table size. The user provides a date up to
// which rows are deleted. The activity log table can get very large very fast so
// clearing old stuff from time to time can be helpful.
func Clear(w http.ResponseWriter, r *http.Request) {
	//date to delete up to
	priorToDate := strings.TrimSpace(r.FormValue("priorToDate"))
	if len(priorToDate) != len("2006-02-02") {
		output.ErrorInputInvalid("Invalid date provided. Date must be in YYYY-MM-DD format and should be a date in the past.", w)
		return
	}
	_, err := time.Parse("2006-01-02", priorToDate)
	if err != nil {
		output.Error(err, "Invalid data provided.", w)
		return
	}

	//Delete.
	rowsDeleted, err := db.ClearActivityLog(r.Context(), priorToDate)
	if err != nil {
		output.Error(err, "Could not clear activity log.", w)
		return
	}

	output.UpdateOKWithData(rowsDeleted, w)
}
