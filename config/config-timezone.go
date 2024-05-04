package config

import (
	"strconv"
	"time"
)

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

// GetTimezoneOffsetForSQLite returns the offset to the Timezone set in the config
// file usable in datetime() "-5 hours" format.
//
// This is typically used in SQL queries when comparing or returning the
// DatetimeCreated column which is stored in the UTC timezone.
func GetTimezoneOffsetForSQLite() (s string) {
	//Get the location based on the config Timezone.
	loc := GetLocation()

	//Get the current time in the location so we can determine the offset of the
	//location versus UTC.
	now := time.Now().In(loc)

	//Determine the offset.
	_, offsetSecs := now.Zone()
	offsetHours := offsetSecs / 60 / 60

	s = strconv.Itoa(offsetHours) + " hours"
	return
}
