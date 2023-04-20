/*
Package licensefile defines the format for a license key file and tooling for
creating, signing, reading, verifying, and interacting with a license key file.

A license key file is a text file storing YAML or JSON encoded data. The data stored
in a license key file has a standardized format that can include customized data per
a third-party app's (app reading and verifying the license) needs. The Signature
field contains a public-private keypair signature of the other publically marshalled
fields in the license key file. The signature authenticates the data in the license
key file; if any data in the license key file is changed, or the signature is changed,
validation will fail.

# Creating and Signing a License Key File

The process of creating a license key file and signing it is as follows:
 1. A File's fields are populated.
 2. The File is marshalled, using FileFormat, to a byte slice.
 3. The bytes are hashed.
 4. The hash is signed via a previously defined private key.
 5. The generated signature is encoded into a textual format.
 6. The human readable signature is set to the File's Signature field.
 7. The File is marshalled, again, but this time with the Signature field populated.
 8. The output from marshalling is saved to a text file or served to as a browser response.

# Reading and Verifying a License Key File

The process of reading and verifying a license key file is as follows:
 1. A text file is read from the filesystem.
 2. The read bytes are unmarshalled to a File struct.
 3. The signature is removed from the File and decoded.
 4. The File is marshalled and the resulting bytes are hashed.
 5. The decoded signature is compared against the hash using a public key.
 6. If the signature is valid, the license key file's data can be used.
 7. Check that the license isn't expired.
*/
package licensefile
