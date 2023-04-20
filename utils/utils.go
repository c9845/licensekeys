/*
Package utils implements helpful funcs that are reused throughout the app
and provide some simply, yet redundantly coded, functionaly.  Think of this
package as storing some "generic" funcs that golang doesn't provide or funcs
that we reuse in a bunch of places.

This package should not import any other packages.  This package should only
be imported into other packages.  This is to prevent import loops.  Plus,
considering that these are basic helper funcs there should be no need to depend
on anything else (maybe the std lib).
*/
package utils

import "time"

// ValidateStartAndEndDate checks if valid start and end dates were provided and that
// the start date is before the end date. This is used for validating filters for
// queries. The defaultRange is used if a start date was not provided to create a
// start date that is defaultRange days in the past.
//
// Inputs and outputs are both yyyy-mm-dd formatted.
func ValidateStartAndEndDate(startDateFromGUI, endDateFromGUI string, equalOkay bool) (startDateForQuery, endDateForQuery, errMsg string, err error) {
	//Start date.
	startDateParsed, err := time.Parse("2006-01-02", startDateFromGUI)
	if err != nil {
		errMsg = "Could not parse start date."
		return
	}

	//End date.
	endDateParsed, err := time.Parse("2006-01-02", endDateFromGUI)
	if err != nil {
		errMsg = "Could not parse start date."
		return
	}

	// log.Println("utils.ValidateStartAndEndDate", "start", "gui:", startDateFromGUI, "parsed:", startDateParsed)
	// log.Println("utils.ValidateStartAndEndDate", "  end", "gui:", endDateFromGUI, "parsed:", endDateParsed)

	//Make sure start is before end.
	if startDateParsed == endDateParsed && equalOkay {
		//this is allowable at times, i.e: when we want to look up a specific day's
		//production or received materials.
	} else if !startDateParsed.Before(endDateParsed) {
		errMsg = "Start date must be before end date."
		return
	}

	startDateForQuery = startDateParsed.Format("2006-01-02")
	endDateForQuery = endDateParsed.Format("2006-01-02")
	return
}
