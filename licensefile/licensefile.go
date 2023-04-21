package licensefile

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

// Errors.
var (
	// ErrBadSignature is returned from Verify() or Verify...() when a File's Signature
	// cannot be verified with the given public key.
	ErrBadSignature = errors.New("signature invalid")

	// ErrMissingExpireDate is returned when trying to check if a license is expires
	// or in how long it expires via the Expired() or ExpiresIn() funcs. This error
	//should really never be returned since the only time these funcs are used are
	//with an existing license's data.
	ErrMissingExpireDate = errors.New("missing expire date")

	//ErrExpired is returned when checking if a license's expire date is in the past.
	//
	//This is ONLY used for the Verify() func that handles signature verification and
	//expiration checking since this func only returns an error, not multiple return
	//values. The Expired() and ExpiresIn() funcs both return a non-error value to
	//represent an expired license.
	ErrExpired = errors.New("license expired")
)

// File defines the format of data stored in a license key file. This is the body of
// the text file.
//
// Struct tags are needed for YAML since otherwise when marshalling the field names
// will be converted to lowercase. We want to maintain camel case since that matches
// the format used when marshalling to JSON.
//
// We use a struct with a map, instead of just map, so that we can more easily interact
// with common fields and store some non-marshalled license data. More simply, having
// a struct is just nicer for interacting with.
type File struct {
	//Optionally displayed fields per app. These are at the top of the struct
	//definition so that they will be displayed at the top of the marshalled data just
	//for ease of human reading of the license key file.
	LicenseID int64  `json:"LicenseID,omitempty" yaml:"LicenseID,omitempty"`
	AppName   string `json:"AppName,omitempty" yaml:"AppName,omitempty"`

	//This data copied from db-license.go and always included in each license key file.
	CompanyName    string `yaml:"CompanyName"`
	ContactName    string `yaml:"ContactName"`
	PhoneNumber    string `yaml:"PhoneNumber"`
	Email          string `yaml:"Email"`
	IssueDate      string `yaml:"IssueDate"`      //YYYY-MM-DD
	IssueTimestamp int64  `yaml:"IssueTimestamp"` //unix timestamp in seconds
	ExpireDate     string `yaml:"ExpireDate"`     //YYYY-MM-DD, in UTC timezone for easiest comparison in DaysUntilExpired()

	//The name and value for each custom field result. This is stored as a key
	//value pair and we use an interface since custom fields can have many types and
	//this is just easier.
	Extras map[string]interface{} `json:"Extras,omitempty" yaml:"Extras,omitempty"`

	//Signature is the result of signing the hash of File (all of the above fields)
	//using the private key. The result is stored here and File is output to a text
	//file known as the complete license key file. This file is distributed to and
	//imported into your app by the end-user to allow the app's use.
	Signature string `yaml:"Signature"`

	//Stuff used for signing or verifying a license file. These are never included in
	//the license key file that is distributed.
	//
	//During verification, these fields are populated just for debugging.
	fileFormat   FileFormat      //the format a file was unmarshaled from.
	readFromPath string          //path a license file was read from.
	publicKey    []byte          //the public key used to verify a license.
	keyPairAlgo  KeyPairAlgoType //algorithm type of the public key.
}

// GenerateKeyPair creates and returns a new private and public key.
func GenerateKeyPair(k KeyPairAlgoType) (private, public []byte, err error) {
	//Make sure a valid key pair type was provided.
	if !slices.Contains(keyPairAlgoTypes, k) {
		err = fmt.Errorf("invalid key pair type, should be one of '%s', got '%s'", keyPairAlgoTypes, k)
		return
	}

	//Generate the key pair.
	switch k {
	case KeyPairAlgoECDSAP256, KeyPairAlgoECDSAP384, KeyPairAlgoECDSAP521:
		private, public, err = GenerateKeyPairECDSA(k)
	case KeyPairAlgoRSA2048, KeyPairAlgoRSA4096:
		private, public, err = GenerateKeyPairRSA(k)
	case KeyPairAlgoED25519:
		private, public, err = GenerateKeyPairED25519()
	}

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

// Read reads a license key file from the given path, unmarshals it, and returns it's
// data as a File. This checks if the file exists and the data is of the correct
// format, however, this DOES NOT check if the license key file itself (the contents
// of the file and the signature) is valid. You should call Verify() on the returned
// File immediately after calling this func.
func Read(path string, format FileFormat) (f File, err error) {
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

	//Unmarshal the file's contents. Upon success, this will set the File's FileFormat
	//field accordingly.
	f, err = Unmarshal(contents, format)

	//If unmarshalling was successful, save the path to the license file since we know
	//the file exists.
	if err == nil {
		f.readFromPath = path
	} else {
		f.readFromPath = "unknown"
	}

	return
}

// Sign creates a signature for a license file. The signature is set in the provided
// File's Signature field. The private key must be decrypted, if needed, prior to
// being provided. The signature will be encoded per the File's EncodingType.
func (f *File) Sign(privateKey []byte, keyPairAlgo KeyPairAlgoType) (err error) {
	err = keyPairAlgo.Valid()
	if err != nil {
		return
	}

	switch keyPairAlgo {
	case KeyPairAlgoECDSAP256, KeyPairAlgoECDSAP384, KeyPairAlgoECDSAP521:
		err = f.SignECDSA(privateKey, keyPairAlgo)
	case KeyPairAlgoRSA2048, KeyPairAlgoRSA4096:
		err = f.SignRSA(privateKey, keyPairAlgo)
	case KeyPairAlgoED25519:
		err = f.SignED25519(privateKey)
	}

	return
}

// hash generates a checksum of the marshalled File's data per the key pair algorithm
// that will be used to sign the hash. This is used as part of the signing process
// since we sign a hash, not the underlying File. This is also used when verifying
// the license key file since we compare the hash against the signature with a public
// key.
func (f *File) hash(keyPairAlgo KeyPairAlgoType) (hash []byte, err error) {
	//Make sure the Signature field is blank prior hashing since if the Signature
	//field is present, it will add a source of randomness and will be replaced
	//anyway by the signature generated within this func.
	f.Signature = ""

	//Encode the struct as bytes per the File's FileFormat. We reuse the FileFormat
	//here since if a third-party app is validating a license file, it already has
	//support for the provided FileFormat (to decode the file's data into a File
	//struct) and reusing the same format for marshalling before hashing just makes
	//sense.
	//
	//We tried marshalling with gob.NewEncoder() and Encode() but this doesn't
	//ignore non-exported struct fields nor fields we don't want included in the
	//license file (i.e.: fields with `json:"-"`).
	b, err := f.Marshal()
	if err != nil {
		err = errors.New(err.Error() + " file format required to marshal data before hashing")
		return
	}

	//Calculate the hash. The hash algorithm is determined by the key pair algorithm.
	err = keyPairAlgo.Valid()
	if err != nil {
		return
	}

	//Use the correct hash algorithm per the key pair algorithm.
	//https://www.rfc-editor.org/rfc/rfc5656#section-6.2.1
	//Default to SHA1 for RSA.
	//Default to SHA512 for ED25519.
	switch keyPairAlgo {
	case KeyPairAlgoECDSAP256:
		h := sha256.Sum256(b)
		hash = []byte(h[:])
	case KeyPairAlgoECDSAP384:
		h := sha512.Sum384(b)
		hash = []byte(h[:])
	case KeyPairAlgoECDSAP521:
		h := sha512.Sum512(b)
		hash = []byte(h[:])
	case KeyPairAlgoRSA2048:
		h := sha1.Sum(b)
		hash = []byte(h[:])
	case KeyPairAlgoRSA4096:
		h := sha1.Sum(b)
		hash = []byte(h[:])
	case KeyPairAlgoED25519:
		h := sha512.Sum512(b)
		hash = []byte(h[:])
	}

	return
}

// encodeSignature returns the generated signature encoded as a string. The returned
// value is the signature that will be set in the File's Signature field.
//
// base64 is used because it will generate shorter signatures than base32 or hex.
func (f *File) encodeSignature(b []byte) {
	f.Signature = base64.StdEncoding.EncodeToString(b)
}

// decodeSignature returns the File's Signature field as a []byte for use when
// verifying the license key file with a public key.
//
// base64 is used because it will generate shorter signatures than base32 or hex.
func (f *File) decodeSignature() (b []byte, err error) {
	b, err = base64.StdEncoding.DecodeString(f.Signature)
	return
}

// VerifySignature checks if a File's signature is valid by checking it against the
// publicKey. This DOES NOT check if a File is expired.
func (f *File) VerifySignature(publicKey []byte, keyPairAlgo KeyPairAlgoType) (err error) {
	err = keyPairAlgo.Valid()
	if err != nil {
		return
	}

	switch keyPairAlgo {
	case KeyPairAlgoECDSAP256, KeyPairAlgoECDSAP384, KeyPairAlgoECDSAP521:
		err = f.VerifySignatureECDSA(publicKey, keyPairAlgo)
	case KeyPairAlgoRSA2048, KeyPairAlgoRSA4096:
		err = f.VerifySignatureRSA(publicKey, keyPairAlgo)
	case KeyPairAlgoED25519:
		err = f.VerifySignatureED25519(publicKey)
	}

	//If verification was successful, save the public key info to the license file
	//since we know the information was correct.
	if err == nil {
		f.publicKey = publicKey
		f.keyPairAlgo = keyPairAlgo
	}

	return
}

// Verify checks if a File's signature is valid and if the license has expired. This
// calls VerifySignature() and Expired().
func (f *File) Verify(publicKey []byte, keyPairAlgo KeyPairAlgoType) (err error) {
	//Verify the signature.
	err = f.VerifySignature(publicKey, keyPairAlgo)
	if err != nil {
		return
	}

	//Check if license is expired.
	expired, err := f.Expired()
	if err != nil {
		return
	} else if expired {
		err = ErrExpired
	}

	return
}

// Expired returns if a lincense File's expiration date is in the past.
//
// You should call Verify() first!
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

// ExpiresIn calculates duration until a license File expires. If a license is
// expired, a negative duration is returned.
//
// You should call Verify() first!
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
