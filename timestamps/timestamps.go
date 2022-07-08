/*
Package timestamps holds helper functions for dates, times, and timestamps. This is
really just used so that we don't have the type the format string over and over and
possibly make a mistake or use a slightly different version of the format.
*/
package timestamps

import (
	"time"
)

//YMDHMS returns a YYYY-MM-DD HH:MM:SS formatted datetime timestamp.
func YMDHMS() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}

//YMD returns a YYYY-MM-DD formatted datetime timestamp.
func YMD() string {
	return time.Now().UTC().Format("2006-01-02")
}
