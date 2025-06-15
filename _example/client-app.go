package main

import (
	"log"

	"github.com/c9845/licensekeys/v4/licensefile"
)

// publicKey is the public part of the key pair generated for your
// application. This should be embedded in your distributed app.
//
// note, no extra newlines or whitespace characters.
const publicKey = "" +
	`-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAvzRdvbuQZOOPHTUT+xacIV1u3rvAPiEDMlkLq7RgibE=
-----END PUBLIC KEY-----
`

// licenseReadFromFile is the contents of a license file generated
// by the license key server using the matching private key for the
// public key.
//
// Note: no extra newlines or whitespace characters.
// Note: indentation is via two spaces, not tabs.
const licenseReadFromFile = "" +
	`{ 
		"LicenseID": 10023,
		"AppName": "Example",
		"CompanyName": "ACME Dynamite",
		"ContactName": "Wyle E Coyote",
		"PhoneNumber": "123-555-1212",
		"Email": "wyle@example.com",
		"IssueDate": "2022-05-07",
		"IssueTimestamp": 1651958341,
		"ExpireDate": "2049-09-21",
		"Data": {,
			"String": "Hello World!",
			"Boolean": true,
			"Integer": 5,
			"Decimal": 5.55
		},
		"Signature": "GBDAEIIAYPGNFZPDUQHMJ2WDQ4NETOLA4EZZVJ2LWVXIRGBZ6SKGMULV3ESAEIIA2QXHQ2HXLSIF7CUWZVLILT4FNKKDXHOLALM5QV3HQV5K4QWMVICQ===="
	}`

func init() {
	//Read the license key from a file.
	//
	//This code is commented out since we have the file's contents stored as a const
	//in this example; we do not have to read the file's data from a file.
	// lic, err := licensefile.Read("/path/to/license.txt")
	// if err != nil {
	// 	log.Fatal("Could not read file.", err)
	// 	return
	// }

	//This code is not necessary when using licencefile.Read(). Read() calls
	//Unmarshal internally. This is strictly here because the license file for this
	//example is stored as a const.
	var lic licensefile.File
	err := lic.Unmarshal([]byte(licenseReadFromFile))
	if err != nil {
		log.Fatalln("Could not unmarshal license file.", err)
		return
	}

	//At this point, you can access the license file's fields, but you do not know if
	//the values are valid/trustworthy since you have not verified the signature
	//against the file's contents using a public key. Do not use the lic for anything
	//at this point!

	//Verify the license file.
	//
	//This compares the signature against the license's data using the public key. If
	//the signature, or any data has in the license file has been altered, the
	//signature will not be valid.
	err = lic.Verify([]byte(publicKey))
	if err == licensefile.ErrBadSignature {
		//License is invalid, signature does not match the license's data. Either the
		//license has been tampered with (data or signature) or an incorrect public
		//key was used for verification.
		log.Fatalln("Signature is invalid.", err)
		return
	} else if err != nil {
		//Some other error occured.
		log.Fatalln("Error verifying license signature.", err)
		return
	}

	// err = lic.VerifyED25519([]byte(publicKey))
	// if err != nil {
	// 	log.Fatalln("Could not verify license.", err)
	// 	return
	// }

	//Now, we know the license is valid and has not been tampered with. We can now
	//trust the data in the license. However, we do not know if the license is
	//expired.

	//Check if the license is expired.
	//
	//Based on your needs, you may handle this elsewhere in your app to handle
	//expiration more gracefully than just prevening the app from running upon seeing
	//an expired license.
	expired, err := lic.Expired()
	if err != nil {
		//Error while checking expire date, should rarely, if ever, occur.
		log.Fatalln("Could not verify expire date.", err)
		return
	}
	if expired {
		//License is expired. Handle as needed.
		log.Fatalf("License is expired, expired on %s.", lic.ExpireDate)
		return
	}

	//We know the license is valid and it has not expired, use the data in the
	//license file as needed. It would be best to store the license file data now
	//for future use elsewhere in the app.
	log.Println("License verified!")
	log.Println("License ID:", lic.LicenseID)

	if i, err := lic.DataAsInt("CustomFieldInt"); err != nil {
		log.Fatalln("Error reading custom field as integer...", err)
		return
	} else {
		log.Println("Extra Int:", i)
	}

	//You will also want to handle reverifying and rechecking the expire date in your
	//app every so often to handle long-running apps. In other words, this code only
	//checks if a license is expired when the app starts, however, if a license
	//expires in the the future, you may want to reduce your app's functionality,
	//display errors, or just stop the app.
}

func main() {
	//Your app code...
}
