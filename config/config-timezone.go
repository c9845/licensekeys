package config

import "time"

// tzLoc is the location used in the time package as determined from the Timezone field
// in the config file. If Timezone in config field is blank, this will be set to UTC.
//
// This is set in Read() and used for displaying dates and times in the app in a
// timezone more friendly than UTC for the end users.
var tzLoc *time.Location

// GetLocation returns the value of the tzLoc variable. This func is used so that the
// tzLoc variable isn't mistakenly overwritten if it were exported.
func GetLocation() *time.Location {
	return tzLoc
}
