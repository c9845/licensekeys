/*
Package config handles configuration of the app. The config data is used for low-level
settings of the app and can be used elsewhere in this app.

The config file is in yaml format for easy readability. However, the file does not
need to end in the yaml extension.

A config file can either be manually created, see template in _documenation, or it can
be created upon first-run of this app if no config file exists at the given path. A
default config will be used if no path to a config file is provided, although this
should not really be used.

This package must not import any other packages from within this app to prevent
import loops (besides minor utility packages).

---

When adding a new field to the config file:
 - Add the field to the File type below.
 - Determine any default value(s) for the field and set it in newDefaultConfig().
 - Document the field in the template config file in the documenation.
 - Set and validation in validate().
 - Determine if the field should be listed in nonPublishedFields.
 - Add the field to the diagnostics page, see pages.Diagnostics().

Try to keep the organization/order of the config fields the same between the config
file templates, the type defined below, validation, and diagnostics.
*/
package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/pwds"
	"github.com/c9845/licensekeys/v2/version"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"

	//Needed to support timezone being set in config and system running this app not
	//having a list of IANA timezones. For example, if running in a docker scratch
	//container.
	_ "time/tzdata"
)

//DefaultConfigFileName is the typical name of the config file.
const DefaultConfigFileName = "licensekeys.conf"

//File defines the list of configuration fields. The value for each field will be set
//by a default or read from a config file. The config file is typically stored in the
//same directory as the executable.
//
//Don't use uints! If user provided a negative number an ugly error message is
//kicked out. We would rather check for the negative number here and provide a nicer
//error message.
type File struct {
	DBPath        string `yaml:"DBPath"`        //The path the the database file.
	DBJournalMode string `yaml:"DBJournalMode"` //Sets the mode for writing to the database file; delete or wal.

	WebFilesStore       string `yaml:"WebFilesStore"`       //Where HTML, CSS, and JS will be sourced and served from; on-disk, on-disk-memory, or embedded.
	WebFilesPath        string `yaml:"WebFilesPath"`        //The absolute path to the directory storing the app's HTML, CSS and JS files.
	FQDN                string `yaml:"FQDN"`                //The domain/subdomain the app serves on and matches your HTTPS certificate, also used for cookies. "." is acceptable but not advised.
	Port                int    `yaml:"Port"`                //The port the app serves on. An HTTPS terminating proxy should redirect port 80 here.
	UseLocalFiles       bool   `yaml:"UseLocalFiles"`       //Serve third-party CSS and JS files from this app's files or from an internet CDN.
	StaticFileCacheDays int    `yaml:"StaticFileCacheDays"` //The number of days a browser will cache the app's CSS and JS files for. -1 disables caching.

	LoginLifetimeHours        float64 `yaml:"LoginLifetimeHours"`        //The time a user will remain logged in for.
	TwoFactorAuthLifetimeDays int     `yaml:"TwoFactorAuthLifetimeDays"` //The time between when a 2FA token will be required. -1 requires it upon each login.

	Timezone                string `yaml:"Timezone"`                //Timezone in IANA format for displaying dates and times.
	MinPasswordLength       int    `yaml:"MinPasswordLength"`       //The shortest length a new password can be.
	PrivateKeyEncryptionKey string `yaml:"PrivateKeyEncryptionKey"` //The key used to encrypt/decrypt the private keys stored in the db. This was if the db is compromised, the keys cannot be used. If not provided, private keys are stored in plaintext. Must be 16, 24, or 32 characters.

	//undocumented, not for end-user usage
	//Make sure each of these fields is in nonPublishedFields to prevent logging.
	Development bool `yaml:"Development"` //shows header in app that app is in development, uses non minified CSS & JSS, enabled some debugging, extra logging, etc.
}

//nonPublishedFields is the list of config fields that we do not advertise to end
//users. These fields are used for development or diagnostic purposes only. The fields
//are listed here so that we can make sure not to print them when the -print-config
//flag is provided to the executable or when we create and write a default config to
//a file..
var nonPublishedFields = []string{
	"Development",
}

//parsedConfig is the data parsed from the config file. This data is stored so that
//we don't need to reparse the config file each time we need a piece of data from it.
//This is not exported so that changes cannot be made to the parsed data as easily.
//Use the Data() func to get the data for use elsewhere.
var parsedConfig File

//Stuff used for validation.
const (
	portMin = 1024
	portMax = 65535

	DBJournalModeRollback = "DELETE"
	DBJournalModeWAL      = "WAL"

	WebFilesStoreOnDisk       = "on-disk"
	WebFilesStoreOnDiskMemory = "on-disk-memory"
	WebFilesStoreEmbedded     = "embedded"
)

var (
	validJournalModes = []string{
		DBJournalModeRollback,
		DBJournalModeWAL,
	}
)

//newDefaultConfig returns a File with default values set for each field.
func newDefaultConfig() (f File, err error) {
	//Get path to executable to build default paths.
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	exeDir := filepath.Dir(exePath)

	f = File{
		DBPath:        filepath.Join(exeDir, "licensekeys.db"),
		DBJournalMode: DBJournalModeRollback, //DELETE is more safe and easier to use in Docker (see Dockerfile).

		WebFilesStore:       WebFilesStoreEmbedded,
		WebFilesPath:        "", //don't need this set since we are using embedded files!
		FQDN:                ".",
		Port:                8007,
		UseLocalFiles:       false,
		StaticFileCacheDays: 7,

		LoginLifetimeHours:        1,
		TwoFactorAuthLifetimeDays: 14,

		Timezone:                "UTC",          //tried using time.Local.String() but this returns "Local" as the timezone which doesn't have much meaning when displayed in the GUI.
		MinPasswordLength:       pwds.MinLength, //the shortest we allow, same as set in pwds package.
		PrivateKeyEncryptionKey: "",             //no encryption by default
	}
	return
}

//Read handles reading and parsing the config file at the provided path. The parsed
//data is sanitized and validated. The print argument is used to print the config as
//it was read/parsed and as it was understood after sanitizing, validating, and
//handling default values.
//
//The parsed configuration is stored in a local variable for access with the Data()
//func. This is done so that the config file doesn't need to be reparsed each time we
//want to get data from it.
func Read(path string, print bool) (err error) {
	// log.Println("Provided config file path:", path, print)

	//Handle path to config file.
	// - If the path is blank, we just use the default config. An empty path should
	//   not ever happen since the flag that provides the path has a default set.
	//   However, we still need this since if the path is empty we cannot save the
	//   default config to a file (we don't know where to save it).
	// - If a path is provided, check that a file exists at it. If a file does not
	//   exist, create a default config at the given path.
	// - If a file at the path does exist, parse it as a config file.
	var tz string
	if strings.TrimSpace(path) == "" {
		log.Println("Using default config; path to config file not provided.")

		//Get default config.
		cfg, innerErr := newDefaultConfig()
		if innerErr != nil {
			return innerErr
		}

		//We don't get a random private key encryption key since if the app is
		//started over and over without a config file path, the encryption key will
		//be different each time and thus the private keys won't be usable.

		//Save the config to this package for use elsewhere in the app.
		parsedConfig = cfg

		//Save timezone for configuring below.
		tz = cfg.Timezone
	} else if _, err = os.Stat(path); os.IsNotExist(err) {
		log.Println("WARNING! (config) Creating default config at:", path)

		//Get default config.
		cfg, innerErr := newDefaultConfig()
		if innerErr != nil {
			return innerErr
		}

		//Get a random key to encrypt key pair private keys when they are stored
		//to the db. We prefer to store private keys encrypted in the db for data
		//security.
		cfg.PrivateKeyEncryptionKey = getRandomEncryptionKey()

		//Save the config to this package for use elsewhere in the app.
		parsedConfig = cfg

		//Save timezone for configuring below.
		tz = cfg.Timezone

		//Save the config to a file.
		innerErr = cfg.write(path)
		if innerErr != nil {
			return
		}

		//Unset the os.IsNotExist error since we created the file.
		err = nil
	} else {
		log.Println("Using config from file:", path)

		//Read the file at the path.
		f, innerErr := os.ReadFile(path)
		if innerErr != nil {
			return innerErr
		}

		//Parse the file as yaml.
		var cfg File
		innerErr = yaml.Unmarshal(f, &cfg)
		if innerErr != nil {
			return innerErr
		}

		//Print the config, if needed, as it was parsed from the file. This logs out
		//the config fields with the user provided data before any validation.
		if print {
			log.Println("***PRINTING CONFIG AS PARSED FROM FILE***")
			cfg.print(path)
		}

		//Validate & sanitize the data since it could have been edited by a human.
		innerErr = cfg.validate()
		if innerErr != nil {
			return innerErr
		}

		//Save the config to this package for use elsewhere in the app.
		parsedConfig = cfg

		//Save timezone for configuring below.
		tz = cfg.Timezone
	}

	//Handle timezone configuration.
	loc, innerErr := time.LoadLocation(tz)
	if innerErr != nil {
		return innerErr
	}
	tzLoc = loc

	//Print the config, if needed, as it was sanitized and validated. This logs out
	//the config as it was understood by the app and some changes may have been made
	//(for example, user provided an invalid value for a field and a default value
	//was used instead). This also prints out the config if it was created or if the
	//config path was blank and a default config was used instead.
	//Always exit at this point since printing config is just for diagnostics.
	if print {
		log.Println("***PRINTING CONFIG AS UNDERSTOOD BY APP***")
		parsedConfig.print(path)
		os.Exit(0)
		return
	}

	return
}

//write writes a config to a file at the provided path.
func (conf *File) write(path string) (err error) {
	//Marshal to yaml.
	y, err := yaml.Marshal(conf)
	if err != nil {
		return
	}

	//Remove non-published fields. Using struct tags "-" does not work for us since
	//we want to be able read these fields from a file if they exist, we just don't
	//want to tell every user about them.
	yamlLinesBefore := strings.Split(string(y), "\n")
	yamlLinesAfter := []string{}
	for _, l := range yamlLinesBefore {
		key, _, _ := strings.Cut(l, ":")

		if slices.Contains(nonPublishedFields, key) {
			continue
		}

		yamlLinesAfter = append(yamlLinesAfter, l)
	}
	y = []byte(strings.Join(yamlLinesAfter, "\n"))

	//Create the file.
	file, err := os.Create(path)
	if err != nil {
		return
	}
	defer file.Close()

	//Add some comments to config file so a human knows it was generated, not written
	//by a human.
	file.WriteString("#Generated config file for licensekeys.\n")
	file.WriteString("#Generated at: " + time.Now().Format(time.RFC3339) + "\n")
	file.WriteString("#Generated by version: " + version.V + "\n")
	file.WriteString("#This file is in YAML format.\n")
	file.WriteString("\n")
	file.WriteString("#***Do not delete this file!***\n")
	file.WriteString("#***Do not change the PrivateKeyEncryptionKey field's value after you have created private keys; any existing private keys will become unusable.!***\n")
	file.WriteString("\n")

	//Write config to file.
	_, err = file.Write(y)
	return
}

//validate handles sanitizing and validation of a config file's data.
func (conf *File) validate() (err error) {
	//Get defaults to use for cases when user provided invalid input.
	defaults, err := newDefaultConfig()
	if err != nil {
		return
	}

	//Database related.
	conf.DBPath = filepath.FromSlash(strings.TrimSpace(conf.DBPath))
	if conf.DBPath == "" {
		conf.DBPath = defaults.DBPath
		log.Println("WARNING! (config) DBPath value was not given, defaulting to " + conf.DBPath + ".")
	}
	_, innerErr := os.Stat(conf.DBPath)
	if os.IsNotExist(innerErr) {
		log.Println("WARNING! (config) File does not exist at DBPath. If database is being deployed, the file will be created.")
	} else if innerErr != nil {
		err = fmt.Errorf("config: could not validate DBPath. %w", err)
		return
	}

	conf.DBJournalMode = strings.TrimSpace(strings.ToUpper(conf.DBJournalMode))
	if conf.DBJournalMode == "" {
		//user did not provide journal mode, use default
		conf.DBJournalMode = defaults.DBJournalMode
	} else if !slices.Contains(validJournalModes, conf.DBJournalMode) {
		//user provided incorrect journal mode, use default
		conf.DBJournalMode = defaults.DBJournalMode
		log.Println("WARNING! (config) An invalid value was provided for DBJournalMode, defaulting to '" + conf.DBJournalMode + "'.")
	}

	//Web server settings.
	switch conf.WebFilesStore {
	case WebFilesStoreOnDisk:
		// log.Println("WARNING! (config) Web files will be served from disk.")
	case WebFilesStoreOnDiskMemory:
		// log.Println("WARNING! (config) Web files will be served from disk, cache busting files will be saved and served from memory.")
	case WebFilesStoreEmbedded:
		// log.Println("WARNING! (config) Web files will be served from embedded.")
	default:
		conf.WebFilesStore = defaults.WebFilesStore
		log.Println("WARNING! (config) No WebFilesStore set, web files will be served from default (" + conf.WebFilesStore + ").")
	}

	conf.WebFilesPath = filepath.FromSlash(strings.TrimSpace(conf.WebFilesPath))
	if conf.WebFilesStore != WebFilesStoreEmbedded {
		if conf.WebFilesPath == "" {
			conf.WebFilesPath = defaults.WebFilesPath
			log.Println("WARNING! (config) WebFilesPath not provided, defaulting to " + conf.WebFilesPath + ".")
		}
		_, err = os.Stat(conf.WebFilesPath)
		if os.IsNotExist(err) {
			err = fmt.Errorf("config: WebFilesPath is invalid, directory could not be found. %w", err)
			return
		} else if err != nil {
			err = fmt.Errorf("config: Could not validate WebFilesPath. %w", err)
			return
		}
	}

	conf.FQDN = strings.TrimSpace(conf.FQDN)
	if conf.FQDN == "" {
		conf.FQDN = defaults.FQDN
		log.Println("WARNING! (config) FQDN not provided, defaulting to \"" + conf.FQDN + "\".")
	}

	if conf.Port == 0 {
		conf.Port = defaults.Port
	}
	if conf.Port < portMin || conf.Port > portMax {
		return errors.New("config: Port must be between " + strconv.Itoa(portMin) + " and " + strconv.Itoa(portMax))
	}

	if conf.StaticFileCacheDays < 0 {
		log.Println("WARNING! (config) StaticFileCacheDays is invalid, caching of static files is disabled. Value should be an integer greater than 0.")
	} else if conf.StaticFileCacheDays == 0 {
		conf.StaticFileCacheDays = defaults.StaticFileCacheDays
	}

	//Login.
	if conf.LoginLifetimeHours <= 0 {
		conf.LoginLifetimeHours = defaults.LoginLifetimeHours
	}

	if conf.TwoFactorAuthLifetimeDays == 0 {
		conf.TwoFactorAuthLifetimeDays = defaults.TwoFactorAuthLifetimeDays
	} else if conf.TwoFactorAuthLifetimeDays < 0 {
		//special case, if a negative number then every time a user logs in they
		//have to provide 2FA token.
		//_ = "" to remove "empty branch" staticcheck linter warning. This branch
		//is here just for the comments to explain why <0 is a special case.
		_ = ""
	}

	//Misc.
	conf.Timezone = strings.TrimSpace(conf.Timezone)
	if conf.Timezone == "" {
		conf.Timezone = defaults.Timezone
		log.Println("WARNING! (config) Timezone not provided, defaulting to " + conf.Timezone + ".")
	}
	//we check if timezone provided is valid in Read() time.LoadLocation(conf.Timezone).

	if conf.MinPasswordLength == 0 {
		conf.MinPasswordLength = defaults.MinPasswordLength
	} else if conf.MinPasswordLength < defaults.MinPasswordLength {
		conf.MinPasswordLength = defaults.MinPasswordLength
		log.Println("WARNING! (config) MinPasswordLength is too short, defaulting to " + strconv.Itoa(conf.MinPasswordLength) + ".")
	}

	if conf.PrivateKeyEncryptionKey == "" {
		log.Println("WARNING! (config) Private key encryption is disabled.")
	} else if len(conf.PrivateKeyEncryptionKey) != 32 {
		err = errors.New("config: PrivateKeyEncryptionKey must be 16, 24, or 32 characters long, 32 is recommended")
		return
	}

	return
}

//print logs out the configuration file. This is used for diagnostic purposes. This
//will show all fields from the File struct, even fields that the provided config file
//omitted (except nonPublishedFields).
func (conf File) print(path string) {
	//Full path to the config file, so if file is in same directory as the executable
	//and -config flag was not provided we still get the complete path.
	pathAbs, _ := filepath.Abs(path)

	log.Println("Path to config file (flag):", path)
	log.Println("Path to config file (absolute):", pathAbs)

	//Print out config file stuff (actually from parsed struct).
	x := reflect.ValueOf(&conf).Elem()
	typeOf := x.Type()
	for i := 0; i < x.NumField(); i++ {
		fieldName := typeOf.Field(i).Name
		value := x.Field(i).Interface()

		if !slices.Contains(nonPublishedFields, fieldName) {
			log.Println(fieldName+":", value)
		}
	}
}

//Data returns the parsed config file data. This is used in other packages to use
//config file data.
func Data() File {
	return parsedConfig
}

//getRandomEncryptionKey returns a string used for encrypting the private key of a key
//pair.
func getRandomEncryptionKey() (encKey string) {
	//By default, use the longest length key.
	const l = 32

	//Generate random string.
	r := make([]byte, l)
	_, err := rand.Read(r)
	if err == nil {
		encKey = base64.StdEncoding.EncodeToString(r)[:l]
		return
	}

	//Error occured with rand.Read, default to using a hash of the current nanosecond
	//time. This isn't as secure but is a good second option versus just returning an
	//error and an empty encryption key.
	t := time.Now().Format(time.RFC3339Nano)
	s := sha256.Sum256([]byte(t))
	encKey = base64.StdEncoding.EncodeToString(s[:])[:l]

	return
}
