/*
Package keypairs handles the public-private keypairs defined for your apps that are
used to sign your license data to create license keys. Keypairs are used to sign your
data for authenticity purposes. The private key should never leave this app while the
public key can be exported for placement in your app's code.
*/
package keypairs

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/c9845/licensekeys/v4/config"
	"github.com/c9845/licensekeys/v4/db"
	"github.com/c9845/licensekeys/v4/keypairs/kpencrypt"
	"github.com/c9845/licensekeys/v4/licensefile"
	"github.com/c9845/licensekeys/v4/users"
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
	loggedInUserID, err := users.GetUserIDFromRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	k.CreatedByUserID = loggedInUserID

	//Generate the key pair public and private key.
	privateKey, publicKey, err := licensefile.GenerateKeypair()
	if err != nil {
		output.Error(err, "Could not generate key pair.", w)
		return
	}

	//Set data for saving to db. We save the private & public keys as strings in the
	//database just for ease of use. We could store as BLOB (sqlite) but string
	//works fine. Plus, we can inspect the private and public keys in the database
	//more easily when they are stored as strings (as long as private key isn't
	//encrypted).
	k.KeypairAlgo = licensefile.KeypairAlgo
	k.FingerprintAlgo = licensefile.FingerprintAlgo
	k.EncodingAlgo = licensefile.EncodingAlgo
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
		encryptedPrivateKey, err := kpencrypt.Encrypt(privateKey, encryptionKey)
		if err != nil {
			output.Error(err, "Could not save key pair. The private key could not be encrypted. Please contact an administrator.", w)
			return
		}

		k.PrivateKey = string(encryptedPrivateKey)
		k.PrivateKeyEncrypted = true
	}

	//Check if this will be the only active keypair for this app, and if it is, mark
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

// Get returns the list of keypairs for an app. You can optionally filter by active
// only.
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
// to sign a license.
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
