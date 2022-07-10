/*
Package apikeys handles generating, revoking, and listing data on api keys. API keys
are used for authenticating automated access to this app.

Only certain endpoints are accessible via an API key, not all of this app's
functionality is accessible via an outside integration. The limitations are for
security purposes and because not all functionality needs to be accessible via a
public API.
*/
package apikeys

import (
	"context"
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

	"github.com/c9845/licensekeys/v2/db"
	"github.com/c9845/licensekeys/v2/users"
	"github.com/c9845/output"
	"github.com/c9845/sqldb/v2"
	"golang.org/x/exp/slices"
)

//keyLength is the length of random part of each API key. This does not include the
//prefix or the separator!
const keyLength = 40

//keyPrefix defines a prefix that gets prepended to each api key so that an API key
//can be more easily be identified versus just an arbitrary string.
//https://github.blog/2021-04-05-behind-githubs-new-authentication-token-formats/
//
//The keySeparator will be used to separate the prefix and the randomly generated
//part of the key.
const keyPrefix = "lks" //License Key Server

//keySeparator is used to separate the keyPrefix from the randomly generated part
//of an API key.
const keySeparator = "_"

//publicEndpoints are the list of URLs a user can access via an API key. This list
//is checked against in middleware to make sure a request using an API key is
//accessing a publicly accessible endpoint.
var publicEndpoints = []string{
	"/api/v1/licenses/add/",
	"/api/v1/licenses/download/",
	"/api/v1/licenses/renew/",
	"/api/v1/licenses/disable/",
}

//ErrNonPublicEndpoint is returned when a request is made via an api key to an
//endpoint that isn't in the list publicEndpoints.
var ErrNonPublicEndpoint = errors.New("api: access denied to non-public endpoint")

//GetAll looks up a list of all API keys. This is used on the manage API keys page.
func GetAll(w http.ResponseWriter, r *http.Request) {
	cols := sqldb.Columns{
		db.TableAPIKeys + ".ID",
		db.TableAPIKeys + ".DatetimeCreated",
		db.TableAPIKeys + ".Active",
		db.TableAPIKeys + ".Description",
		db.TableAPIKeys + ".K",
		db.TableUsers + ".Username AS CreatedByUsername",
	}

	keys, err := db.GetAPIKeys(r.Context(), true, cols)
	if err != nil {
		output.Error(err, "Could not look up API keys.", w)
		return
	}

	output.DataFound(keys, w)
}

//Generate creates a new API key and saves it to the database.
func Generate(w http.ResponseWriter, r *http.Request) {
	//Get input data.
	raw := r.FormValue("data")

	//Parse.
	var a db.APIKey
	err := json.Unmarshal([]byte(raw), &a)
	if err != nil {
		output.Error(err, "Could not parse data to generate API Key.", w)
		return
	}

	//Validate.
	a.Description = strings.TrimSpace(a.Description)
	if len(a.Description) == 0 {
		output.ErrorInputInvalid("You must provide a description for this API Key.", w)
		return
	}

	//Check if a key with this description already exists and is active. We don't
	//want multiple API keys with the same description since that would make
	//identifying which API key was used more difficult.
	_, err = db.GetAPIKeyByDescription(r.Context(), a.Description)
	if err == nil {
		output.ErrorAlreadyExists("This description is already in use by another API Key. Please use a different description.", w)
		return
	} else if err != nil && err != sql.ErrNoRows {
		output.Error(err, "Could not verify if this Description is already in use.", w)
		return
	}

	//Get user who is creating this new API key.
	loggedInUserID, err := users.GetUserIDByRequest(r)
	if err != nil {
		output.Error(err, "Could not determine the user making this request.", w)
		return
	}
	a.CreatedByUserID = loggedInUserID

	//Generate and save new API key. This is done in a loop so we can handle the rare
	//case of a duplicate key being generated. Duplicate keys will be rejected by the
	//database due to the unique constraint on the column that store the API key.
	for i := 0; i < 5; i++ {
		//Generate a new API key.
		a.K = generateKey(r.Context(), a.Description)

		//Try saving the key to the db. If a duplicate key exists, it will be
		//rejected by the database and this for loop will retry by generating a new,
		//different API key (since generateKey using a timestamp as part of the seed
		//to generate the API key and the timestamp will change between loops of this
		//for block).
		err := a.Insert(r.Context())
		if err != nil && strings.Contains(err.Error(), "Duplicate entry") {
			log.Println("Duplicate API Key generated, trying again...", a.K)
			continue

		} else if err != nil {
			output.Error(err, "Could not create and save new api key.", w)
			return

		} else {
			//API key was saved, it is not a duplicate, exit loop.
			break
		}
	} //end for: generate API key.

	output.InsertOKWithData(a, w)
}

//generateKey actually creates a new API key. An API key is generated using a seed of
//a salt, the user provided description, and a timestamp to add some randomness.
func generateKey(ctx context.Context, apiKeyDesc string) (key string) {
	const salt = "xkr8NVwLg$@ENvPj*S&k"

	//Data to use as seed to generate api key
	hashInputItems := []string{
		salt,                //used so that input to hash isn't just data about the key from the database.
		apiKeyDesc,          //user provided description of the api key.
		time.Now().String(), //we use time stamp to provide additional randomness so that if user tries to create keys with the same description, the keys won't match.
	}

	//Generate hash. The hash, trimmed if needed, will be the API key (minus the
	//prepended API key prefix). The random part of the API key is provided as all
	//upper case characters to remove confusion between upper- and lower-case
	//letters.
	sum := sha256.Sum256([]byte(strings.Join(hashInputItems, "")))
	hash := strings.ToUpper(hex.EncodeToString(sum[:]))

	if len(hash) > keyLength {
		hash = hash[:keyLength]
	}

	//Prepend the prefix. Prefix is used as-is (lower, upper, or mixed case).
	//Underscore separates prefix and hash for easily distinguishing between the
	//two parts.
	key = buildCompleteAPIKey(hash)
	return
}

//buildCompleteAPIKey builds an API key by prepending the prefix and separator.
func buildCompleteAPIKey(partialKey string) string {
	return keyPrefix + keySeparator + partialKey
}

//Revoke marks an API key as inactive.
func Revoke(w http.ResponseWriter, r *http.Request) {
	//Get input.
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)

	//Validate.
	if id < 1 {
		output.ErrorInputInvalid("Could not determine which API key you want to revoke.", w)
		return
	}

	//Mark the key as inactive.
	err := db.RevokeAPIKey(r.Context(), id)
	if err != nil {
		output.Error(err, "Could not revoke API key.", w)
		return
	}

	output.UpdateOK(w)
}

//IsPublicEndpoint checks if a provided URL is in the list of publically accessible
//endpoints. If not, it returnes an error.
func IsPublicEndpoint(urlPath string) bool {
	return slices.Contains(publicEndpoints, urlPath)
}

//KeyLength returns the length of API keys generated inclusive of the key's prefix
//and key separator. This is used during validation of API requests to simply check
//if the provided api key is the correct length before looking up the key in the
//database.
func KeyLength() int {
	return len(keyPrefix) + len(keySeparator) + keyLength
}
