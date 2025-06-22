package licensefile

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"strings"
	"time"
)

// Errors when validating a license key file.
var (
	// ErrBadSignature is returned from Verify() or Verify...() when a File's Signature
	// cannot be verified with the given public key.
	ErrBadSignature = errors.New("signature invalid")

	// ErrMissingExpirationDate is returned when trying to check if a license is expires
	// or in how long it expires via the Expired() or ExpiresIn() funcs. This error
	//should really never be returned since the only time these funcs are used are
	//with an existing license's data.
	ErrMissingExpirationDate = errors.New("missing expire date")
)

// File is the format of a license key file. This is the contents of the textual
// license file.
type File struct {
	//Optionally displayed fields per app's settings (see db-apps.go). These are at
	//the top of the struct definition so that they will be displayed at the top of
	//the text file just for ease of a human reading of the license key file.
	LicenseID string `json:",omitempty"` //UUID, the PublicID.
	AppName   string `json:",omitempty"`

	//This data copied from db-license.go and is always included in each license file.
	CompanyName    string
	ContactName    string
	PhoneNumber    string
	Email          string
	IssueDate      string //YYYY-MM-DD
	ExpirationDate string //YYYY-MM-DD, in UTC timezone for easiest comparison in DaysUntilExpired()

	//Data is any optional data that you want to store in a license file. This
	//map can store anything, and is typically used for storing information that
	//enables certain functionality within your app, for example, a maximum user
	//count.
	//
	//Called "custom fields" when interfacing with this app's database.
	Data map[string]any `json:",omitempty"`

	//Signature is the result of signing a fingerprint (hash) of File using a private
	//key.
	//
	//This value is added to a File before it is written to an actual license file.
	//When verifying a license file, make sure to strip this value out first.
	//
	//Omitempty is needed so that "Signature": "" isn't set when calculating
	//fingerprint; this makes calculating fingerprint manually a bit more intuitive.
	Signature string `json:",omitempty"`

	//Info used for debugging.
	readFromPath string `json:"-"` //path a license file was read from.
}

// GenerateKeypair creates and return a new private and public key pair.
func GenerateKeypair() (privateKey, publicKey []byte, err error) {
	//Generate key pair.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return
	}

	//Encode for storage in database and serving private key to users in GUI.
	//
	//Using hex mostly so we don't have to deal with odd padding errors caused by
	//base64 (StdEncoding vs RawStdEncoding...somehow padding still gets used, or
	//not ignored, at times). Plus, we use hex for encoding the fingerprint, so we
	//don't need to add another dependency.
	privateKey = make([]byte, hex.EncodedLen(len(priv)))
	hex.Encode(privateKey, priv)

	publicKey = make([]byte, hex.EncodedLen(len(pub)))
	hex.Encode(publicKey, pub)

	//Used to use PEM encoding, PKCS#8 and PKIX here. Moved to hex encoding for ease
	//of not having to decode PEM all the time.  Maybe make format configurable?

	return
}

// calculateFingerprint hashes File's data after marshalling File to JSON. []bytes
// are returned for use in Sign() and Verify().
//
// Hash algorithm is SHA512.
//
// This uses a COPY of the File since we need to remove the Signature field prior to
// hashing and we don't want to modify the original File so that if we are calculating
// the fingerprint on an already signed license, the Signature will be kept.
func (f *File) calculateFingerprint() (fingerprint []byte, err error) {
	//Make sure the Signature field is empty. The Signature is never included in a
	//fingerprint, the Signature is based upon the fingerprint.
	f.Signature = ""

	//Encode File as JSON, just like it would be encoded in an actual license file.
	b, err := f.marshal()
	if err != nil {
		return
	}

	//Calculate the fingerprint, as []byte.
	h := sha512.Sum512(b)
	fingerprint = h[:]
	return
}

// CalculateFingerprint hashes File's data after marshalling File to JSON. A string
// is returned.
//
// Hash algorithm is SHA512 and the result is hex encoded.
//
// This uses a COPY of the File since we need to remove the Signature field prior to
// hashing and we don't want to modify the original File so that if we are calculating
// the fingerprint on an already signed license, the Signature will be kept.
//
// When comparing the fingerprint generated here to a fingerprint generated using a
// tool such as Cyberchef, make sure the "Rounds" is set to 160.
func (f File) CalculateFingerprint() (fingerprint string, err error) {
	//Calculate the fingerprint, as []byte.
	fpb, err := f.calculateFingerprint()

	//Encode.
	//
	//Hex encoding is used since that is the standard representation of SHA-512. See
	//sha256sum or related commandline tools.
	fingerprint = hex.EncodeToString(fpb[:])

	return
}

// Sign creates a signature for File by signing the File's fingerprint. The File's
// Signature field will be populated since it needs to be included when the File is
// marshalled and written to a textual license file.
//
// The private key must be decrypted, if needed, prior to being provided.
func (f *File) Sign(privateKey []byte) (err error) {
	//Create fingerprint of File.
	fingerprint, err := f.calculateFingerprint()
	if err != nil {
		return
	}

	//Decode private key.
	priv := make([]byte, hex.DecodedLen(len(privateKey)))
	hex.Decode(priv, privateKey)

	//Sign.
	sig := ed25519.Sign(priv, fingerprint)

	//Encode the signature in a human readable format.
	sigTxt := hex.EncodeToString(sig)

	//Set the signature in the license file.
	f.Signature = sigTxt
	return
}

// Write writes a File to out. This is used to output the complete license key file.
// This can be used to write the File to a buffer, as is done when creating a license
// key file, write the File back to the browser as html, or write the File to an actual
// filesystem file.
//
// For use with a buffer:
//
//	//b := bytes.Buffer{}
//	//err := f.Write(&b)
//
// Writing to an http.ResponseWriter:
//
//	//func handler(w http.ResponseWriter, r *http.Request) {
//	//  //...
//	//  err := f.Write(w)
//	//}
func (f *File) Write(out io.Writer) (err error) {
	//Marshal to bytes.
	b, err := f.marshal()
	if err != nil {
		return
	}

	//Write.
	_, err = out.Write(b)
	return
}

// FromFile reads a license file from a file and parses it into a File. The file at
// the given path must have contents in JSON format.
func FromFile(path string) (f File, err error) {
	//Check if a file exists at the provided path.
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return
	}

	//Read the file at the provided path.
	contents, err := os.ReadFile(path)
	if err != nil {
		return
	}

	//Unmarshal the file's contents.
	err = f.unmarshal(contents)
	if err != nil {
		return
	}

	//Save the path to the license file since we know the file exists. This is used
	//for debugging.
	f.readFromPath = path
	return
}

// FromBytes reads a license file from a []byte, for when a license file was read from
// a text file already, was stored in bytes in a database column, or another time. The
// []byte must represent JSON.
func FromBytes(b []byte) (f File, err error) {
	err = f.unmarshal(b)
	return
}

// FromString reads a license file from a string, for when a license file was read from
// a test file already, was stored in a textual database column, or another time. The
// string must represent JSON.
func FromString(s string) (f File, err error) {
	//Unmarshal the file's contents.
	err = f.unmarshal([]byte(s))
	return
}

// Verify checks if a File is a valid license by checking the signature against the
// File's other data using the publicKey.
//
// This DOES NOT check if a File is expired. You should call Expired() on the File
// after calling this func.
//
// Signature verification and expiration date checking were kept separate on purpose
// so that each step can be handled more deliberately with specific handling of
// invalid states (i.e.: for more graceful handling).
//
// This uses a COPY of the File since we need to remove the Signature field prior to
// hashing and verification but we don't want to modify the original File so it can
// be used as it was parsed/unmarshalled.
//
// Verifying with a third-party tool:
//  1. https://cyphr.me/ed25519_tool/ed.html
//     - Msg Encoding: hex.
//     - Message: the fingerprint.
//     - Key Encoding: hex.
//     - Public key: the public key.
//     - Signature: the signature.
func (f File) Verify(publicKey []byte) (err error) {
	//Strip the signature out of the File since the signature is not based upon itself.
	sigTxt := f.Signature
	f.Signature = ""

	//Decode the signature.
	sig, err := hex.DecodeString(sigTxt)
	if err != nil {
		return
	}

	//Create fingerprint of File.
	fingerprint, err := f.calculateFingerprint()
	if err != nil {
		return
	}

	//Decode public key.
	pub := make([]byte, hex.DecodedLen(len(publicKey)))
	hex.Decode(pub, publicKey)

	//Verify.
	valid := ed25519.Verify(pub, fingerprint, sig)
	if !valid {
		err = ErrBadSignature
		return
	}

	//Signature is valid.
	return
}

// marshal encodes the File as JSON.
//
// This func is defined so we don't need to use json.MarshalIndent everywhere and so
// that we can be certain we are using the same marshalling everywhere (calculating
// fingerprint and writing File to a text file).
func (f *File) marshal() (b []byte, err error) {
	b, err = json.MarshalIndent(f, "", "  ")
	return
}

// unmarshal decodes JSON into File.
//
// This func is defined so we don't need to use json.Unmarshall everywhere and so
// that we can be certain we are using the same unmarshalling everywhere (reading
// File from a file, []byte, or string).
func (f *File) unmarshal(b []byte) (err error) {
	err = json.Unmarshal(b, &f)
	return
}

// Expired returns if a lincense File's expiration date is in the past.
//
// You should only call this AFTER calling VerifySignature() otherwise the expiration
// date in the File is untrustworthy and could have been modified.
//
// Signature verification and expiration date checking were kept separate on purpose
// so that each step can be handled more deliberately with specific handling of
// invalid states (i.e.: for more graceful handling).
func (f *File) Expired() (yes bool, err error) {
	//Make sure a expiration data is provided. It should always be provided since
	//you would call this func after reading a license file and verifying it's
	//signature.
	if strings.TrimSpace(f.ExpirationDate) == "" {
		return false, ErrMissingExpirationDate
	}

	//Check if license is expired.
	expDate, err := time.Parse("2006-01-02", f.ExpirationDate)
	if err != nil {
		return
	}

	yes = expDate.Before(time.Now())
	return
}

// ExpiresIn calculates duration until a license File expires. The returned duration
// will be negative for an expired license.
//
// You should only call this AFTER calling VerifySignature() otherwise the expiration
// date in the File is untrustworthy and could have been modified.
func (f *File) ExpiresIn() (d time.Duration, err error) {
	//Make sure a expiration data is provided. It should always be provided since
	//you would call this func after reading a license file and verifying it's
	//signature.
	if strings.TrimSpace(f.ExpirationDate) == "" {
		return 0, ErrMissingExpirationDate
	}

	//Get duration until license is expired.
	expDate, err := time.Parse("2006-01-02", f.ExpirationDate)
	if err != nil {
		return
	}

	d = time.Until(expDate)
	return
}

// ExpiresInDays is a wrapper around ExpiresIn that returns the number of days a
// license File expires in. The returned days will be negative for an expired
// license.
//
// You should only call this AFTER calling VerifySignature() otherwise the expiration
// date in the File is untrustworthy and could have been modified.
func (f *File) ExpiresInDays() (days int, err error) {
	//Get duration license will expire in.
	dur, err := f.ExpiresIn()
	if err != nil {
		return
	}

	//Convert to days.
	days = int(math.Floor(dur.Hours() / 24))
	return
}
