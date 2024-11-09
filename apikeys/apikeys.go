/*
Package apikeys handles generating, revoking, and listing data on api keys.

API keys are used for authenticating automated access to this app. In other words,
other apps accessing this app's data.

The app can store and use multiple API keys. Idealy one API key is used for each
integration to this app. Doing so allows for revoking one API key without affecting
other integrations.

Only certain endpoints are accessible via an API key. Not all of this app's data
is accessible via an outside integration. This is for security purposes.  The list
of accessible endpoints is noted below in the publicEndpoints slice.

API keys are stored in plain text on the server. This is done since is someone outside
of and approved user of an API key has access to the API key, they can already perform
actions of that API key. API keys are not like passwords where they are often reused
or provided each time. Furthermore, if someone has access to a list of API keys then
they most likely have access to the database anyway. The best use case for a hashed
value being stored in the database is that someone browsing the database won't be
able to use an API key just by looking at the stored value.
*/
package apikeys

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v3/db"
	"github.com/c9845/licensekeys/v3/users"
	"github.com/c9845/output"
)

// keyLength is the length of random part of the api key that is generated randomly
// for each key. This does not include the prefix. 40 was chosen as it is the length
// of an SHA256 hash hex encoded.
const keyLength = 40

// keyPrefix defines a prefix that gets prepended to each api key so that
// the key can be more easily identified versus just an arbitrary hash.
// https://github.blog/2021-04-05-behind-githubs-new-authentication-token-formats/
const keyPrefix = "lks_"

// GetAll looks up a list of all API keys.
func GetAll(w http.ResponseWriter, r *http.Request) {
	//Get data.
	keys, err := db.GetAPIKeys(r.Context(), true)
	if err != nil {
		output.Error(err, "Could not look up api keys.", w)
		return
	}

	output.DataFound(keys, w)
}

// Generate creates a new API key and saves it to the database.
func Generate(w http.ResponseWriter, r *http.Request) {
	//Get input data.
	raw := r.FormValue("data")

	//Parse into struct.
	var a db.APIKey
	err := json.Unmarshal([]byte(raw), &a)
	if err != nil {
		output.Error(err, "Could not parse data to generate API Key.", w)
		return
	}

	//Sanitize.
	a.Description = strings.TrimSpace(a.Description)

	//Validate.
	a.Description = strings.TrimSpace(a.Description)
	if len(a.Description) == 0 {
		output.ErrorInputInvalid("You must provide a description for this API Key.", w)
		return
	}

	//Check if a key with this description already exists and is active.
	_, err = db.GetAPIKeyByDescription(r.Context(), a.Description)
	if err == nil {
		output.ErrorAlreadyExists("This Description is already in use by another API Key. Please use a different description.", w)
		return
	} else if err != sql.ErrNoRows {
		output.Error(err, "Could not verify if this Description is already in use.", w)
		return
	}

	//Get user ID of logged in user who is creating this API key.
	loggedInUserID, err := users.GetUserIDFromRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	a.CreatedByUserID = loggedInUserID

	//Generate the new API key.
	//
	//This is done in a loop so we can handle cases where a duplicate key is created,
	//even though a duplicate key being created should rarely, if ever, happen.
	maxAttempts := 5
	for i := 0; i < maxAttempts; i++ {
		//Generate a new API key.
		a.K = generateKey(a.Description)

		//Try saving the key to the db. If a duplicate key exists, it will be
		//rejected by the database and this for loop will retry by generating a new,
		//different API key (since generateKey using a timestamp as part of the seed
		//to generate the API key and the timestamp will change between loops of this
		//for block).
		err := a.Insert(r.Context())
		if err != nil && (strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "UNIQUE constraint failed")) {
			//Duplicate entry = MariaDB (MariaDB usage is very experimental).
			//UNIQUE constraint failed = SQLite.

			log.Println("Duplicate API Key generated, trying again...", a.K)
			continue

		} else if err != nil {
			output.Error(err, "Could not create and save new API key.", w)
			return

		} else {
			//API key was saved, it is not a duplicate, exit loop.
			break
		}
	} //end for: generate API key.

	//Make sure a non-duplicate key was generated and saved. The for loop will exit
	//without a key being saved if maxAttempts duplicates were generated.
	if a.ID == 0 {
		output.Error(errors.New("too many attempts generating API key"), "An API key could not be generated, too many duplicate keys were created. Try again or contact an administrator.", w)
		return
	}

	output.InsertOK(a.ID, w)
}

// generateKey generates a new API key from a seed text.
func generateKey(apiKeyDesc string) (key string) {
	//Salt just helps to add some length to the seed and some extra randomness and
	//so that we aren't just hashing data stored in the database in plain text.
	const salt = "da7J3nkKJ^dsvd-A23hk"

	//Data to use as seed to generate api key
	hashInputItems := []string{
		salt,                //used so that input to hash isn't just data about the key from the database.
		apiKeyDesc,          //user provided description of the api key.
		time.Now().String(), //we use time stamp to provide additional randomness so that if user tries to create keys with the same description, the keys won't match.
	}

	//Generate hash.
	hashInput := strings.Join(hashInputItems, "")
	sum := sha256.Sum256([]byte(hashInput))
	hash := hex.EncodeToString(sum[:])

	//Trim the length of the key if needed.
	if len(hash) > keyLength {
		hash = hash[:keyLength]
	}

	//Prepend the prefix.
	//
	//Prefix is used as provided (lower, upper, or mixed case).
	//The hash is all uppercased to reduce confusion between characters.
	key = keyPrefix + strings.ToUpper(hash)
	return
}

// Revoke marks an API key as inactive. An inactive API key cannot be reactivated.
func Revoke(w http.ResponseWriter, r *http.Request) {
	//Get input.
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)

	//Validate.
	if id < 1 {
		output.ErrorInputInvalid("Could not determine which API key you want to revoke.", w)
		return
	}

	//Mark the API key as inactive.
	err := db.RevokeAPIKey(r.Context(), id)
	if err != nil {
		output.Error(err, "Could not revoke API key.", w)
		return
	}

	output.UpdateOK(w)
}

// KeyLength returns the length of API keys generated inclusive of the key prefix.
// This is used during validation of API requests to simply check if the provided API
// key is the correct length before looking up the API key in the database.
func KeyLength() int {
	return len(keyPrefix) + keyLength
}

// Update saves changes to an API key. Only the API key's description and permissions
// can be changed. The actual key can never be changed.
func Update(w http.ResponseWriter, r *http.Request) {
	//Get input data.
	raw := r.FormValue("data")

	//Parse into struct.
	var a db.APIKey
	err := json.Unmarshal([]byte(raw), &a)
	if err != nil {
		output.Error(err, "Could not parse data to generate API Key.", w)
		return
	}

	//Sanitize.
	a.Description = strings.TrimSpace(a.Description)

	//Validate.
	if len(a.Description) == 0 {
		output.ErrorInputInvalid("You must provide a description for this API Key.", w)
		return
	}

	//Check if a key with this description already exists and is active.
	existing, err := db.GetAPIKeyByDescription(r.Context(), a.Description)
	if err != nil && err != sql.ErrNoRows {
		output.Error(err, "Could not look up if an API Key with description already exists.", w)
		return
	} else if a.ID != existing.ID {
		output.ErrorInputInvalid("An API Key with this description already exists.", nil)
		return
	}

	//Save.
	err = a.Update(r.Context())
	if err != nil {
		output.Error(err, "Could not update API Key.", w)
		return
	}

	output.UpdateOK(w)
}

type apiKeyContextKeyType string

// APIKeyContextKey is the name of the key that stores an API key's ID in the request
// context. This is used to save the API Key ID in middleware-externalAPI.go and is
// used to get teh API Key ID via context.Value().
const APIKeyContextKey apiKeyContextKeyType = "api-key-id"
