package main

import (
	"log"

	"github.com/c9845/licensekeys/v2/licensefile"
)

//publicKey is the public part of the key pair generated for your
//application. This should be embedded in your distributed app.
//
//note, no extra newlines or whitespace characters.
const publicKey = "" +
	`-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAvzRdvbuQZOOPHTUT+xacIV1u3rvAPiEDMlkLq7RgibE=
-----END PUBLIC KEY-----
`

//licenseReadFromFile is the contents of a license file generated
//by the license key server using the matching private key for the
//public key.
//
//Note: no extra newlines or whitespace characters.
//Note: indentation is via two spaces, not tabs.
const licenseReadFromFile = "" +
	`LicenseID: 10001
AppName: App1
CompanyName: ACME Dynamite
ContactName: Wyle E Coyote
PhoneNumber: 123-555-1212
Email: wyle@example.com
IssueDate: "2022-05-30"
IssueTimestamp: 1653941696
ExpireDate: "2023-05-30"
Extras:
  CustomFieldInt: 50
  CustomFieldString: Hello World
Signature: K6GyCUnfJLLc4LEE98UFH4kPGUF5rzhqvXrTUYtk4Ut1wq/LPWE3E1CSMfosm6pcGhNIrNKQLtNekrybK4h3Cg==
`

//Per your configured settings when you created the app, key pair, and license.
const (
	fileFormat       = licensefile.FileFormatYAML
	keyPairAlgorithm = licensefile.KeyPairAlgoED25519
)

func init() {
	//Read the license key from a file. The file's data is unmarshalled.
	//
	//The code immediately below reads a license file's data from a filesystem file
	//and unmarshals it into a File. However, the code is commented out since we have
	//the file's contents stored as a const; we do not have to read the file's data
	//from a file.
	// licenseData, err := licenseFile.Read("/path/to/license.txt", fileFormat)
	// if err != nil {
	// 	log.Fatal("Could not read file.", err)
	// 	return
	// }

	//Parse the file's contents per the expected file format. This unmarhals the
	//file's contents as YAML or JSON. The format was set in the license key server
	//app before the license was created.
	licenseData, err := licensefile.Unmarshal([]byte(licenseReadFromFile), fileFormat)
	if err != nil {
		log.Fatalln("Could not unmarshal license file.", err)
		return
	}

	//At this point, you can access the license file's fields, but you do not know if
	//the values are valid (haven't been tampered with) since you have not verified
	//the signature. Do not use the licenseData for anything!

	//Verify the license file. This compares the signature against the license's
	//other data (other fields) via the public key. If the signature, or any data is
	//altered in the license file, the signature will not be valid and this will
	//return false.
	//
	//The keyPairAlgorithm (RSA, ECDSA, ED25519) is needed to use the correct
	//function for handling the signature. You could use one of the Verify...()
	//functions instead.
	err = licenseData.Verify([]byte(publicKey), keyPairAlgorithm)
	if err != nil {
		log.Fatalln("Could not verify license.", err)
		//os.Exit(1)
		return
	}

	// err = licenseData.VerifyED25519([]byte(publicKey))
	// if err != nil {
	// 	log.Fatalln("Could not verify license.", err)
	// 	return
	// }

	//The license was verified, use the data as needed. Since the signature was
	//validated, we can be assured that any data used from the license file is
	//authentic and was the data set when the license was created. It would be best
	//to store the licenseData value now for use elsewhere in the app.
	log.Println("License verified!")
	log.Println("License ID:", licenseData.LicenseID)

	if i, err := licenseData.ExtraAsInt("CustomFieldInt"); err != nil {
		log.Fatalln("Error reading custom field as integer...", err)
		return
	} else {
		log.Println("Extra Int:", i)
	}
}

func main() {
	//Your app code...
}
