/*
Package utils implements helpful funcs that are reused throughout the app
and provide some simply, yet redundantly coded, functionaly.

This package should not import any other packages.  This package should only
be imported into other packages.  This is to prevent import loops.  Plus,
considering that these are basic helper funcs there should be no need to depend
on anything else (maybe the std lib).
*/
package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// RandString generates a random string. This is used for creating login or 2FA
// cookie values. This retuns a string formated in base64.
func RandString(length int) (s string, err error) {
	b := make([]byte, length)
	_, err = rand.Read(b)
	if err != nil {
		return
	}

	//format as base64
	s = base64.StdEncoding.EncodeToString(b)

	if len(s) > length {
		s = s[:length]
	}
	return
}
