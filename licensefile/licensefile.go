package licensefile

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
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

// File defines the format of data stored in a license key file. This is the body of
// the text file.
//
// We use a struct with a map, instead of just map, so that we can more easily interact
// with common fields and store some non-marshalled license data. More simply, having
// a struct is just nicer for interacting with.
type File struct {
	//Optionally displayed fields per app. These are at the top of the struct
	//definition so that they will be displayed at the top of the marshalled data just
	//for ease of human reading of the license key file.
	LicenseID int64  `json:"omitempty"`
	AppName   string `json:",omitempty"`

	//This data copied from db-license.go and always included in each license key file.
	CompanyName    string
	ContactName    string
	PhoneNumber    string
	Email          string
	IssueDate      string //YYYY-MM-DD
	IssueTimestamp int64  //unix timestamp in seconds
	ExpireDate     string //YYYY-MM-DD, in UTC timezone for easiest comparison in DaysUntilExpired()

	//Metadata is any optional data that you want to store in a license file. This
	//field can store anything, and is typically used for storing information that
	//enables certain functionality within your app. For example, a maximum user
	//count.
	//
	//Called "custom fields" when interfacing with the database. Previously called
	//"Metadata" when interfacing with a license File. "Metadata" just sounded ugly.
	Metadata map[string]any `json:",omitempty"`

	//Signature is the result of signing the hash of File (all of the above fields)
	//using the private key. The result is stored here and File is output to a text
	//file known as the complete license key file. This file is distributed to and
	//imported into your app by the end-user to allow the app's use.
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

// Fingerprint returns the checksum/hash of a license's data in a human readable
// format. This is typically used when activating a license.
//
// The fingerprint is returned as base64 encoded SHA512.
func (f *File) Fingerprint() (fingerprint string, err error) {
	//Calculate the fingerprint as a []byte.
	b, err := f.fingerprint()
	if err != nil {
		return
	}

	//Encode.
	fingerprint = base64.StdEncoding.EncodeToString(b)
	return
}

// Fingerprint returns the checksum/hash of a license's data as a []byte. This is used
// when signing or verifying a license.
//
// The fingerprint is returned as base64 encoded SHA512.
func (f *File) fingerprint() (fingerprint []byte, err error) {
	//Make sure the Signature field is blank prior hashing since the Signature is
	//based upon the fingerprint.
	f.Signature = ""

	//Encode the struct as bytes.
	b, err := f.Marshal()
	if err != nil {
		err = fmt.Errorf("file format required to marshal data before hashing, %w", err)
		return
	}

	//Hash the license.
	h := sha512.Sum512(b)
	fingerprint = []byte(h[:])
	return
}

// Sign creates a signature for a license file. The signature is set in the provided
// File's Signature field. The private key must be decrypted, if needed, prior to
// being provided. The signature will be encoded in base64.
func (f *File) Sign(privateKey []byte) (err error) {
	//Create fingerprint of license. This is what we actually sign.
	fingerprint, err := f.fingerprint()
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

// Verify checks if a File's signature is valid by checking it against the license's
// fingerprint using the public key.
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
	//Decode the signature.
	sig, err := base64.StdEncoding.DecodeString(f.Signature)
	if err != nil {
		return
	}

	//Remove signature from license so we can hash the license file's data. Signature
	//is based on all data in license file but itself!
	f.Signature = ""

	//Create fingerprint of license. This is what we actually sign.
	fingerprint, err := f.fingerprint()
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
	b, err := f.Marshal()
	if err != nil {
		return
	}

	//Write.
	_, err = out.Write(b)
	return
}

// Marshal encodes the File as JSON.
func (f *File) Marshal() (b []byte, err error) {
	b, err = json.MarshalIndent(f, "", "  ")
	return
}

// Unmarshal decodes JSON into File.
func (f *File) Unmarshal(b []byte) (err error) {
	err = json.Unmarshal(b, &f)
	return
}

// Read reads a license key file from the given path, unmarshals it, and returns it's
// data as a File. This checks if the file exists and the data is of the correct
// format.
//
// If you do not need to read a license from a file, unmarshal the license data into
// a File or create a File in some other manner, then call Verify().
//
// This DOES NOT check if the license key file itself (the contents of the file and
// the signature) is valid nor does this check if the license is expired. You should
// call Verify() and Expired() on the returned File immediately after calling this
// func.
func Read(path string) (f File, err error) {
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
	err = f.Unmarshal(contents)
	if err != nil {
		return
	}

	//Save the path to the license file since we know the file exists. This is used
	//for debugging.
	f.readFromPath = path
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
