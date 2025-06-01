/*
Package cookieutils handles setting and getting cookies.
*/
package cookieutils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/c9845/licensekeys/v4/config"
)

var (
	//hmacSecret is used to calculated HMAC signatures. This is populated the by
	//init().
	hmacSecret = make([]byte, 64)

	//once tracks that the hmacSecret was populated exactly once.
	once sync.Once

	//hmacHash is the hash algorithm used for HMAC.
	hmacHash = sha256.New

	//hmacHashSize is the size of the hash.
	hmacHashSize = sha256.Size

	//hmacEncoding is the encoding to use for HMAC.
	hmacEncoding = base64.StdEncoding
)

// Set calls http.SetCookie after appending an HMAC signature to the cookie
// Value.
func Set(w http.ResponseWriter, cookie http.Cookie) (err error) {
	//Calculate HMAC of cookie value.
	sig := sign(cookie.Value)
	sigString := hmacEncoding.EncodeToString(sig)

	//Append the signature to the cookie value.
	cookie.Value = cookie.Value + "-" + sigString

	//Write to the cookie ot the response.
	http.SetCookie(w, &cookie)
	return
}

// Read gets the value from a named cookie.
func Read(r *http.Request, name string) (value string, err error) {
	//Get cookie from request.
	cookie, err := r.Cookie(name)
	if err != nil {
		return
	}

	//Decode the cookie's value.
	//
	//Cookies are set as {value}-{hmac}.
	parts := strings.Split(cookie.Value, "-")
	if len(parts) != 2 {
		err = errors.New("unknown cookie format")
		return
	}

	value = parts[0]
	sigString := parts[1]

	//Confirm the HMAC signature.
	sig, err := hmacEncoding.DecodeString(sigString)
	if err != nil {
		return
	}
	if len(sig) != hmacHashSize {
		err = errors.New("invalid cookie signature, wrong lenth")
		return
	}

	expectedSig := sign(value)
	if !hmac.Equal(sig, expectedSig) {
		err = errors.New("invalid cookie signature")
		return
	}

	return
}

// sign creates an HMAC signature of the input.
func sign(s string) (sig []byte) {
	if config.Data().Development {
		//Use hardcoded HMAC secret in development so we don't get logged out each
		//time app is recompiled.
		hmacSecret = []byte("qwertyuiopasdfghjklzxcvbnm1234567890")

	} else {
		once.Do(createHMACSecret)
	}

	//Make sure HMAC key has been set.
	if len(hmacSecret) == 0 {
		log.Fatalln("cookies: hmac secret must be set")
		return
	}

	//Calculate HMAC of cookie value.
	mac := hmac.New(hmacHash, hmacSecret)
	mac.Write([]byte(s))
	sig = mac.Sum(nil)
	return
}

// createHMACSecret populates hmacSecret with a random value.
func createHMACSecret() {
	_, err := rand.Read(hmacSecret)
	if err != nil {
		log.Println("cookies: Could not populate HMAC secret. App will still function, just less securely.", err)
	}
}
