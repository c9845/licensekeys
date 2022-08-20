/*
Package keypairs handles the public-private keypairs defined for your apps that are
used to sign your license data to create license keys. Keypairs are used to sign your
data for authenticity purposes. The private key should never leave this app while the
public key can be exported for placement in your app's code.
*/
package keypairs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/c9845/licensekeys/v2/config"
	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/licensefile"
	"github.com/c9845/licensekeys/v2/users"
	"github.com/c9845/output"
)

// Add saves a new keypair. The keypair data provided is used to generate a new keypair
// which is then saved to the database.
func Add(w http.ResponseWriter, r *http.Request) {
	//Get input data.
	raw := r.FormValue("data")

	//Parse data into struct.
	var k db.KeyPair
	err := json.Unmarshal([]byte(raw), &k)
	if err != nil {
		output.Error(err, "Could not parse data to add key pair.", w)
		return
	}

	//Make sure this isn't being called with an already existing key pair.
	if k.ID != 0 {
		output.ErrorAlreadyExists("Could not determine if you are adding or updating an key pair.", w)
		return
	}

	//Validate.
	errMsg, err := k.Validate(r.Context())
	if err != nil && errMsg != "" {
		output.Error(err, errMsg, w)
		return
	} else if err != nil {
		output.Error(err, "Could not validate data to save key pair.", w)
		return
	} else if errMsg != "" {
		output.ErrorInputInvalid(errMsg, w)
		return
	}

	//Get user who is adding this key pair.
	loggedInUserID, err := users.GetUserIDByRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	k.CreatedByUserID = loggedInUserID

	//Generate the key pair public and private key.
	privateKey, publicKey, err := licensefile.GenerateKeyPair(k.AlgorithmType)
	if err != nil {
		output.Error(err, "Could not generate key pair.", w)
		return
	}

	//Set data for saving to db. We save the private & public keys as strings in the
	//database just for ease of use. We could store as BLOB (sqlite) instead but string
	//works fine. Plus, we can inspecte the private and public keys in the database
	//more easily when they are stored as strings (as long as private key isn't
	//encrypted).
	k.PrivateKey = string(privateKey)
	k.PublicKey = string(publicKey)

	//Check if the private key should be encrypted when stored in the db. This is based
	//upon an encryption key being provided in the config file. If a key is provided,
	//then the private key will be encrypted in the db. Encrypting the private key
	//protects against unauthorized, or compromised, access to the database (as long
	//as the config file which stores the encryption password isn't also compromised).
	//We already checked if the encryption key is the correct length (16, 24, or 32
	//characters) when we validated the config file, however, an error will be returned
	//here if the length is incorrect.
	if len(config.Data().PrivateKeyEncryptionKey) > 0 {
		encryptionKey := config.Data().PrivateKeyEncryptionKey
		encryptedPrivateKey, err := encryptPrivateKey(encryptionKey, privateKey)
		if err != nil {
			output.Error(err, "Could not save key pair. The private key could not be encrypted. Please contact an administrator.", w)
			return
		}

		k.PrivateKey = hex.EncodeToString(encryptedPrivateKey)
		k.PrivateKeyEncrypted = true
	}

	//Check if this will be the only active license for this app, and if it is, mark
	//it as the default.
	kps, err := db.GetKeyPairs(r.Context(), k.AppID, true)
	if err != nil {
		//No returning error since this isn't an end of the world scenario.
		log.Println("keypairs.Add", "could not look up existing keypairs to set default", err)
	}
	if len(kps) == 0 {
		k.IsDefault = true
	}

	//Save.
	err = k.Insert(r.Context())
	if err != nil {
		output.Error(err, "Could not save key pair.", w)
		return
	}

	//Return full data for new key pair. We need this to show the public key and set
	//the "Set As Default" gui state correctly.
	output.InsertOKWithData(k, w)
}

// encryptPrivateKey encrypts a private key with the encryption key provided in the
// config file. This performs AES encryption. This returns a []byte since the input
// unencrypted data is also a []byte; we just keep the type the same for ease of use
// elsewhere.
//
// The encryption key must be 16, 24, or 32 characters long. The encryptionKey is a
// string since that is how it is stored in the config file and we can handle the
// conversion inside this func as needed.
//
// The resulting encryptedPrivateKey includes the nonce, at the start of the
// encrypted data, since it is needed to decrypt the data.
//
// https://pkg.go.dev/crypto/cipher#example-NewGCM-Encrypt
func encryptPrivateKey(encryptionKey string, unencryptedPrivateKey []byte) (encryptedPrivateKey []byte, err error) {
	//Not using a salt here since the encryptionKey must be exactly 32 characters
	//long and we tell the user this when setting the key in the config file. Plus,
	//a salt isn't really helpful since this codebase is open source and the salt
	//could be found easily.

	block, err := aes.NewCipher([]byte(encryptionKey))
	if err != nil {
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	nonce := make([]byte, aesgcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return
	}

	//the nonce will be included at the beginning of the encrypted private key
	encryptedPrivateKey = aesgcm.Seal(nonce, nonce, unencryptedPrivateKey, nil)
	return
}

// DecryptPrivateKey decrypts the private key stored in the database using the
// encryption key provided in the config file. This returns a []byte since the input
// unencrypted data is also a []byte; we just keep the type the same for ease of use
// elsewhere.
//
// The encryption key must be 16, 24, or 32 characters long. The encryptionKey is a
// string since that is how it is stored in the config file and we can handle the
// conversion inside this func as needed.
//
// This func is only used when signing a newly created license file. Once a license
// key file has been created, the signature is stored in the db and we just retrieve
// it when a license key file needs to be downloaded.
//
// https://pkg.go.dev/crypto/cipher#example-NewGCM-Decrypt
func DecryptPrivateKey(encryptionKey string, encryptedPrivateKey []byte) (unecryptedPrivKey []byte, err error) {
	//Not using a salt here since the encryptionKey must be exactly 32 characters
	//long and we tell the user this when setting the key in the config file. Plus,
	//a salt isn't really helpful since this codebase is open source and the salt
	//could be found easily.

	block, err := aes.NewCipher([]byte(encryptionKey))
	if err != nil {
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	//get the nonce from the beginning of the encrypted private key
	//reset the encrypted private key to not include the nonce
	l := aesgcm.NonceSize()
	nonce, encryptedPrivateKey := encryptedPrivateKey[:l], encryptedPrivateKey[l:]
	unecryptedPrivKey, err = aesgcm.Open(nil, nonce, encryptedPrivateKey, nil)
	return
}

// Get returns the list of keypairs. You can optionally filter by active only.
func Get(w http.ResponseWriter, r *http.Request) {
	appID, _ := strconv.ParseInt(r.FormValue("appID"), 10, 64)
	activeOnly, _ := strconv.ParseBool(r.FormValue("activeOnly"))

	if appID < 1 {
		output.ErrorInputInvalid("Could not determine which app you want to look up key pairs for.", w)
		return
	}

	items, err := db.GetKeyPairs(r.Context(), appID, activeOnly)
	if err != nil {
		output.Error(err, "Could not get list of key pairs.", w)
		return
	}

	output.DataFound(items, w)
}

// Delete marks a keypair as inactive. The keypair will no longer be available for use
// to sign a license. Old licenses will still use their assigned keypair even when
// keypair is deleted.
func Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)

	if id < 1 {
		output.ErrorInputInvalid("Could not determine which key pair you want to delete.", w)
		return
	}

	k := db.KeyPair{
		ID: id,
	}
	err := k.Delete(r.Context())
	if err != nil {
		output.Error(err, "Could not delete key pair.", w)
		return
	}

	output.UpdateOK(w)
}

// Default marks a keypair as the default for the app. This marks all other keypairs
// for this app as non-default.
func Default(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)

	if id < 1 {
		output.ErrorInputInvalid("Could not determine which key pair you want to mark as default.", w)
		return
	}

	//Set the default keypair. This will also set all other keypairs as non-default
	//to make sure only one keypair is marked as default for the app.
	k := db.KeyPair{
		ID: id,
	}
	err := k.SetIsDefault(r.Context())
	if err != nil {
		output.Error(err, "Could not set key pair as default.", w)
		return
	}

	output.UpdateOK(w)
}
