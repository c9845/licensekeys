package licensefile

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

//File defines the format of data stored in a license key file. This is the body of
//the text file.
//
//Struct tags are needed for YAML since otherwise when marshalling the field names
//will be converted to lowercase. We want to maintain camel case since that matches
//the format used when marshalling to JSON.
//
//We use a struct with a map, instead of just map, so that we can more easily interact
//with common fields and store some non-marshalled license data. More simply, having
//a struct is just nicer for interacting with.
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

	//Stuff used for signing, verifying, or reverifying a license file. These are never
	//included in the license key file that is distributed.
	//
	//During verification, these fields are populated so that they can be reused if a
	//license file needs to be reverified. Doing so makes reverification easier since
	//the user implementing reverification doesn't need to provide this info again (it
	//was already provided during initial verification of the license).
	fileFormat   FileFormat      //the format a file was unmarshaled from.
	readFromPath string          //path a license file was read from.
	publicKey    []byte          //the public key used to verify a license.
	keyPairAlgo  KeyPairAlgoType //algorithm type of the public key.
}

//SetFileFormat populates the fileFormat field. This func is needed since the
//fileFormat field is not exported since it is not distributed/written in a license
//file.
func (f *File) SetFileFormat(format FileFormat) {
	f.fileFormat = format
}

//FileFormat returns a File's fileFormat field. This func is needed since the
//fileFormat field is not exported since it is not distributed/writted in a license
//file.
func (f *File) FileFormat() FileFormat {
	return f.fileFormat
}

//GenerateKeyPair creates and returns a new private and public key.
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

//Write writes a File to out. This is used to output the complete license key file.
//This can be used to write the File to a buffer, as is done when creating a license
//key file, write the File back to the browser as html, or write the File to an actual
//filesystem file.
//
//For use with a buffer:
//  //b := bytes.Buffer{}
//  //err := f.Write(&b)
//
//Writing to an http.ResponseWriter:
//  //func handler(w http.ResponseWriter, r *http.Request) {
//  //  //...
//  //  err := f.Write(w)
//  //}
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

//Sign creates a signature for a license file. The signature is set in the provided
//File's Signature field. The private key must be decrypted, if needed, prior to
//being provided. The signature will be encoded per the File's EncodingType.
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
		err = f.SignED25519(privateKey, keyPairAlgo)
	}

	return
}

//ErrBadSignature is returned from Verify() or Verify...() when a File's Signature
//cannot be verified.
var ErrBadSignature = errors.New("signature invalid")

//Verify checks if a File is valid. This checks the signature against the File's
//contents using the provided public key.
func (f *File) Verify(publicKey []byte, keyPairAlgo KeyPairAlgoType) (err error) {
	err = keyPairAlgo.Valid()
	if err != nil {
		return
	}

	switch keyPairAlgo {
	case KeyPairAlgoECDSAP256, KeyPairAlgoECDSAP384, KeyPairAlgoECDSAP521:
		err = f.VerifyECDSA(publicKey, keyPairAlgo)
	case KeyPairAlgoRSA2048, KeyPairAlgoRSA4096:
		err = f.VerifyRSA(publicKey, keyPairAlgo)
	case KeyPairAlgoED25519:
		err = f.VerifyED25519(publicKey, keyPairAlgo)
	}

	//If verification was successful, save the public key info to the license file
	//since we know the information was correct. We use this in the ReverifyEvery func.
	if err == nil {
		f.publicKey = publicKey
		f.keyPairAlgo = keyPairAlgo
	}

	return
}

//ErrMissingExpireDate is returned when trying to calculate the DaysUntilExpired but
//no expire date is set for the license file. This should never happen since the only
//time the DaysUntilExpired func is used is when using a license's data and the license
//file was already created and validated.
var ErrMissingExpireDate = errors.New("missing expire date")

//DaysUntilExpired calculates the days from now until when a license will be expired.
func (f *File) DaysUntilExpired() (diffDays int, err error) {
	if strings.TrimSpace(f.ExpireDate) == "" {
		return 0, ErrMissingExpireDate
	}

	now := time.Now().UTC().Truncate(24 * time.Hour)
	expires, err := time.Parse("2006-01-02", f.ExpireDate)
	if err != nil {
		return
	}

	diff := expires.Sub(now)
	diffHours := diff.Hours()
	diffDays = int(diffHours / 24)

	return
}

//Read reads a license key file from the given path, unmarshals it, and returns it's
//data as a File. This checks if the file exists and the data is of the correct
//format, however, this DOES NOT check if the license key file itself (the contents
//of the file and the signature) is valid. You should call Verify() on the returned
//File immediately after calling this func.
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
	//the file exists. We use this in the ReverifyEvery func.
	if err == nil {
		f.readFromPath = path
	} else {
		f.readFromPath = "unknown"
	}

	return
}

//hash generates a checksum of the marshalled File's data per the key pair algorithm
//that will be used to sign the hash. This is used as part of the signing process
//since we sign a hash, not the underlying File. This is also used when verifying
//the license key file since we compare the hash against the signature with a public
//key.
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

//encodeSignature returns the generated signature encoded as a string. The returned
//value is the signature that will be set in the File's Signature field.
//
//base64 is used because it will generate shorter signatures than base32 or hex.
func (f *File) encodeSignature(b []byte) (err error) {
	f.Signature = base64.StdEncoding.EncodeToString(b)
	return
}

//decodeSignature returns the File's Signature field as a []byte for use when
//verifying the license key file with a public key.
//
//base64 is used because it will generate shorter signatures than base32 or hex.
func (f *File) decodeSignature() (b []byte, err error) {
	b, err = base64.StdEncoding.DecodeString(f.Signature)
	return
}

//Reverify performs reverfication of a license. This can be used to make sure a
//license file hasn't been tampered with. This must be called after Verify() has been
//called because Verify() saves some information to the File to remove the need for
//arguments being passed to this func.
//
//Typically you will want to run reverification every so often. You may want to wrap
//this func in a for/while loop that runs every "x" minutes/hours/days.
//Example:
//  //_ = f.Verify(...)
//  //
//  //go func() {
//  //  for {
//  //    time.Sleep(1*time.Hour)
//  //    err := f.Reverify()
//  //    if err != nil {
//  //      log.Println("License file invalid upon reverification.")
//  //      //do something about tampered with license,: os.Exit(1), log.Fatal(...), etc.
//  //    }
//  //  }
//  //}()
func (f *File) Reverify() (err error) {
	//Reread license from file as the stored path.
	fr, err := Read(f.readFromPath, f.fileFormat)
	if err != nil {
		log.Println("reverify path", f.readFromPath)
		return
	}

	err = fr.Verify(f.publicKey, f.keyPairAlgo)
	return
}
