/*
Package pwds implements functionality for creating a secure hash of a password and
for verifying a password matches a stored hash.
*/
package pwds

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// MinLength is the shortest password we allow.
const MinLength = 10

// bcryptCost is the work factor, higher is more resistant to attacks.
const bcryptCost = 13

// ErrBadPassword is used when an invalid password is given. This is used so we don't
// send back the bcrypt error message.
var ErrBadPassword = errors.New("bad password")

// Create gets a hash from a plaintext password.
func Create(password string) (string, error) {
	passwordByte := []byte(password)

	hash, err := bcrypt.GenerateFromPassword(passwordByte, bcryptCost)
	return string(hash), err
}

// IsValid validates a cleartext password against its possible hash.
func IsValid(password, hash string) (bool, error) {
	passwordByte := []byte(password)

	err := bcrypt.CompareHashAndPassword([]byte(hash), passwordByte)
	if err == bcrypt.ErrMismatchedHashAndPassword {
		//password does not match hash
		return false, ErrBadPassword
	} else if err != nil {
		//some other error occured
		return false, err
	}

	//password matches hash
	return true, nil
}
