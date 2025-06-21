package licensefile

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
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

	// ErrMissingExpireDate is returned when trying to check if a license is expires
	// or in how long it expires via the Expired() or ExpiresIn() funcs. This error
	//should really never be returned since the only time these funcs are used are
	//with an existing license's data.
	ErrMissingExpireDate = errors.New("missing expire date")
)

// Reference information.
const (
	//KeypairAlgo is the algorithm used to generate a public-private key pair that
	//signs and verifies license files. ed25519 was chosen because it results in the
	//shortest signatures.
	KeypairAlgo = "ed25519"

	//FingerprintAlgo is the algorithm used to hash the license file data prior to
	//signing it. sha512 was chosen because it is modern, does not have proven
	//weaknesses, and well supported.
	FingerprintAlgo = "sha512"

	//EncodingAlgo is the method of encoding the fingerprint and signature into a
	//human-readbale alphabet that can be used in text files or otherwise. Base64 was
	//chosen because is results in the shortest signature.
	EncodingAlgo = "base64"

	//FileFormat is the format of the data in a license file. JSON was chosen because
	//it is supported by the golang standard library and has good representation of
	//numbers, strings, and booleans.
	FileFormat = "json"
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
	IssueTimestamp int64  //unix timestamp in seconds
	ExpireDate     string //YYYY-MM-DD, in UTC timezone for easiest comparison in DaysUntilExpired()

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
	Signature string

	//Info used for debugging.
	readFromPath string `json:"-"` //path a license file was read from.
}

// GenerateKeypair creates and return a new private and public key pair.
func GenerateKeypair() (private, public []byte, err error) {
	//Generate key pair.
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return
	}

	//Encode the private key.
	x509PrivateKey, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return
	}
	pemBlockPrivateKey := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509PrivateKey,
	}
	private = pem.EncodeToMemory(pemBlockPrivateKey)

	//Encode the public key.
	x509PublicKey, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return
	}
	pemBlockPublicKey := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: x509PublicKey,
	}
	public = pem.EncodeToMemory(pemBlockPublicKey)

	return
}

// calculateFingerprint hashes File's data after marshalling File to JSON. []bytes
// are returned for use in Sign() and Verify().
//
// Hash algorithm is SHA512.
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
func (f *File) CalculateFingerprint() (fingerprint string, err error) {
	//Calculate the fingerprint, as []byte.
	fpb, err := f.calculateFingerprint()

	//Encode.
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

	//Decode the private key for use. This is not decrypting the private key!
	pemBlock, _ := pem.Decode(privateKey)
	x509Key, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return
	}

	//Sign the fingerprint.
	sig := ed25519.Sign(x509Key.(ed25519.PrivateKey), fingerprint[:])

	//Encode the signature in a human readable format.
	sigTxt := base64.StdEncoding.EncodeToString(sig)

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
// This uses a COPY of the File since need to remove the Signature field prior to
// hashing and verification but we don't want to modify the original File so it can
// be used as it was parsed/unmarshalled.
func (f File) Verify(publicKey []byte) (err error) {
	//Strip the signature out of the File since the signature is not based upon itself.
	sigTxt := f.Signature
	f.Signature = ""

	//Decode the signature.
	sig, err := base64.StdEncoding.DecodeString(sigTxt)
	if err != nil {
		return
	}

	//Create fingerprint of File.
	fingerprint, err := f.calculateFingerprint()
	if err != nil {
		return
	}

	//Decode the public key.
	pemBlock, _ := pem.Decode(publicKey)
	x509Key, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return
	}

	//Verify the signature.
	valid := ed25519.Verify(x509Key.(ed25519.PublicKey), fingerprint[:], sig)
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
	if strings.TrimSpace(f.ExpireDate) == "" {
		return false, ErrMissingExpireDate
	}

	//Check if license is expired.
	expDate, err := time.Parse("2006-01-02", f.ExpireDate)
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
	if strings.TrimSpace(f.ExpireDate) == "" {
		return 0, ErrMissingExpireDate
	}

	//Get duration until license is expired.
	expDate, err := time.Parse("2006-01-02", f.ExpireDate)
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
