/*
Package config handles configuration of the app. The configuration can simply be
a default, or a config file can be provided. The config data is used for low-level
settings of the app and can be used elsewhere in this app.

The config file is in YAML format for easy readability. However, the file does not
need to end in the yaml extension. YAML is used since it allows for adding comments
to the config file (JSON does not). Having comments is really nice for providing some
additional info in the config file.

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
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v4/version"
	"gopkg.in/yaml.v3"

	//Needed to support timezone being set in config and system running this app not
	//having a list of IANA timezones. For example, if running in a docker scratch
	//container.
	_ "time/tzdata"
)

// DefaultConfigFileName is the typical name of the config file.
const DefaultConfigFileName = "licensekeys.conf"

// File defines the list of configuration fields. The value for each field will be set
// by a default or read from a config file. The config file is typically stored in the
// same directory as the executable.
//
// Don't use uints! If user provided a negative number an ugly error message is
// kicked out. We would rather check for the negative number here and provide a nicer
// error message.
type File struct {
	DBPath        string `yaml:"DBPath"`        //The path the the database file.
	DBJournalMode string `yaml:"DBJournalMode"` //Sets the mode for writing to the database file; delete or wal.

	WebFilesStore string `yaml:"WebFilesStore"` //Where HTML, CSS, and JS will be sourced and served from; on-disk, on-disk-memory, or embedded.
	WebFilesPath  string `yaml:"WebFilesPath"`  //The absolute path to the directory storing the app's HTML, CSS and JS files.
	UseLocalFiles bool   `yaml:"UseLocalFiles"` //Serve third-party CSS and JS files from this app's files or from an internet CDN.
	Host          string `yaml:"Host"`          //The host the app listens on. Default is 127.0.0.1, aka localhost. Set to server's IP, or 0.0.0.0, to be able to access app directly on host:port without a proxy.
	Port          int    `yaml:"Port"`          //The port the app serves on. An HTTPS terminating proxy should redirect port 80 here.

	LoginLifetimeHours        float64 `yaml:"LoginLifetimeHours"`        //The time a user will remain logged in for.
	TwoFactorAuthLifetimeDays int     `yaml:"TwoFactorAuthLifetimeDays"` //The time between when a 2FA token will be required. -1 requires it upon each login.

	Timezone                string `yaml:"Timezone"`                //Timezone in IANA format for displaying dates and times.
	MinPasswordLength       int    `yaml:"MinPasswordLength"`       //The shortest length a new password can be.
	PrivateKeyEncryptionKey string `yaml:"PrivateKeyEncryptionKey"` //The key used to encrypt/decrypt the private keys stored in the db. This was if the db is compromised, the keys cannot be used. If not provided, private keys are stored in plaintext. Must be 16, 24, or 32 characters.

	//undocumented, not for end-user usage
	//Make sure each of these fields is in nonPublishedFields to prevent logging.
	Development bool `yaml:"Development"` //shows header in app that app is in development, uses non minified CSS & JSS, enabled some debugging, extra logging, etc.
}

// nonPublishedFields is the list of config fields that we do not advertise to end
// users. These fields are used for development or diagnostic purposes only. The fields
// are listed here so that we can make sure not to print them when the -print-config
// flag is provided to the executable or when we create and write a default config to
// a file..
var nonPublishedFields = []string{
	"Development",
}

// parsedConfig is the data parsed from the config file. This data is stored so that
// we don't need to reparse the config file each time we need a piece of data from it.
// This is not exported so that changes cannot be made to the parsed data as easily.
// Use the Data() func to get the data for use elsewhere.
var parsedConfig File

// Stuff used for validation.
const (
	portMin = 1024
	portMax = 65535

	DBJournalModeRollback = "DELETE"
	DBJournalModeWAL      = "WAL"

	WebFilesStoreOnDisk   = "on-disk"
	WebFilesStoreEmbedded = "embedded"
)

var (
	validJournalModes = []string{
		DBJournalModeRollback,
		DBJournalModeWAL,
	}
)

// Errors.
var (
	//ErrNoFilePathGiven is returned when trying to parse the config file but no
	//file path was given.
	ErrNoFilePathGiven = errors.New("config: no file path given")
)

// newDefaultConfig returns a File with default values set for each field.
func newDefaultConfig() (f File, err error) {
	//Get path to where binary is being run from to build default paths.
	workingDir, err := os.Getwd()
	if err != nil {
		return
	}

	//Paths are always forward slashed in YAML so random parsing errors don't occur
	//(like "did not find expected hexdecimal number" when path string is surrounded
	//by double quotes, on Windows).
	dbPath := filepath.ToSlash(filepath.Join(workingDir, "licensekeys.db"))

	f = File{
		DBPath:        dbPath,                //
		DBJournalMode: DBJournalModeRollback, //DELETE is more safe and easier to use in Docker (see Dockerfile).

		WebFilesStore: WebFilesStoreEmbedded, //embedded means less files to distribute
		WebFilesPath:  "",                    //not needed for default embedded file, not set to make config file cleaner and less confusing.
		UseLocalFiles: true,                  //prefer our distributed files, prevents issues with CDNs.
		Host:          "127.0.0.1",           //Listen within localhost only.
		Port:          8007,                  //

		LoginLifetimeHours:        1,  //just a safe default.
		TwoFactorAuthLifetimeDays: 14, //just a safe default.

		Timezone:                "UTC", //tried using time.Local.String() but this returns "Local" as the timezone which doesn't have much meaning when displayed in the GUI.
		MinPasswordLength:       10,    //the shortest we allow, same as set in pwds package.
		PrivateKeyEncryptionKey: "",    //no encryption by default
	}
	return
}

// Read handles reading and parsing the config file at the provided path. The parsed
// data is sanitized and validated.
//
// The parsed configuration is stored in a local variable for access with the
// Data() func. This is done so that the config file doesn't need to be reparsed
// each time we want to get data from it.
//
// If the path is blank, a default in-app/in-memory config is used. If the path is
// provided but a file does not exist, a default config if written to the file at the
// path.
//
// The print argument is used to print the config as it was read/parsed and as it was
// understood after sanitizing, validating, and handling default values.
func Read(path string, print bool) (err error) {
	//Clean the path to remove anything odd.
	path = filepath.Clean(path)

	//Get absolute directory path of file in path, since path could be relative, for
	//displaying in logging. Absolute path is nicer since it makes finding a config
	//file easier.
	absPath, _ := filepath.Abs(path)

	//Handle path to config file.
	var cfg File
	if strings.TrimSpace(path) == "" {
		//The path provided for the config file flag is blank (-config=""). This
		//should never really happen since the flag defines a default value (a file
		//named licensekeys.conf in the working directory where binary is being run from).
		//This simply catches instances where user provides -config="" for some odd
		//reason. Since a path was not provided, we cannot save a config file
		//anywhere. In this case, we just use an "in memory" config that uses default
		//values.
		log.Println("WARNING! (config) Using built-in default config; path to config file was not provided.")

		//Get default config.
		defaultConfig, innerErr := newDefaultConfig()
		if innerErr != nil {
			return innerErr
		}
		cfg = defaultConfig

		//We don't get a random private key encryption key since if the app is
		//started over and over without a config file path, the encryption key will
		//be different each time and thus the private keys won't be usable.

	} else if _, err = os.Stat(path); os.IsNotExist(err) {
		//A path was provided to the config file flag but a file does not exist at
		//the given path. A config file will be saved at the provided path with
		//default values.
		//
		//The path provided to the -config flag could be a user provided value or a
		//default value (i.e.: flag wasn't provided at all).
		log.Println("WARNING! (config) Config does not exist, creating default config at:", absPath)

		//Get default config.
		defaultConfig, innerErr := newDefaultConfig()
		if innerErr != nil {
			return innerErr
		}
		cfg = defaultConfig

		//Get a random key to encrypt key pair private keys when they are stored
		//to the db. We prefer to store private keys encrypted in the db for data
		//security.
		cfg.PrivateKeyEncryptionKey = getRandomEncryptionKey()

		//Save the default config to the file noted in the provided path.
		innerErr = cfg.write(path)
		if innerErr != nil {
			return innerErr
		}

		//Unset the os.IsNotExist error since we created the file.
		err = nil
	} else {
		//A path was provided to the config file flag and a file exists at the given
		//path. Parse the file as a config file. If the file isn't a valid config
		//file, error out, otherwise continue running the app.
		log.Println("Using config from file:", absPath)

		//Read the file at the path.
		f, innerErr := os.ReadFile(path)
		if innerErr != nil {
			return innerErr
		}

		//Parse the file as yaml.
		innerErr = yaml.Unmarshal(f, &cfg)
		if innerErr != nil {
			return innerErr
		}

		//Print the config, if needed, as it was parsed from the file. This logs
		//out the config fields with the user provided data before any validation.
		if print {
			log.Println("***PRINTING CONFIG AS PARSED FROM FILE***")
			cfg.print(path)
		}
	}

	//Validate & sanitize the data since it could have been edited by a human. We do
	//this here, not just for a config file read from a file, so we can catch any
	//mistakes when a default config is used.
	err = cfg.validate()
	if err != nil {
		return err
	}

	//Create the directories for storing files, if needed.
	err = cfg.createDirectories()

	//Save the config to this package for use elsewhere in the app.
	parsedConfig = cfg

	//Handle timezone configuration.
	loc, innerErr := time.LoadLocation(cfg.Timezone)
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

// write writes a config to a file at the provided path.
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
	file.WriteString("#Generated at: " + time.Now().UTC().Format(time.RFC3339) + ".\n")
	file.WriteString("#Generated by version: " + version.V + ".\n")
	file.WriteString("#This file is in YAML format.\n")
	file.WriteString("\n")
	file.WriteString("#***Do not delete this file!***\n")
	file.WriteString("#***Do not change the PrivateKeyEncryptionKey field's value after you have created private keys; any existing private keys will become unusable.!***\n")
	file.WriteString("\n")
	file.WriteString("#***On Windows, when providing a path, use forward slashes in place of a back slashes and surround the path in double quotes!***\n")
	file.WriteString("\n")

	//Write config to file.
	_, err = file.Write(y)
	return
}

// validate handles sanitizing and validation of a config file's data. Validation
// will attempt to a default value if the config file does not contain a field. If
// a field is required and a default is not available, an error will be returned.
//
// Rules for showing messages/logging out stuff:
//  1. If a user/config file did not provide a field and we use a default value, DO
//     NOT tell the user.
//     - Example: DBName.
//  2. If a user/config file provided an invalid value for a field where we have a
//     default value defined and available, TELL the user we are using the default.
//     - EXAMPLE: Port.
//     - Template: log.Printf("WARNING! (config) Port is invalid...")
//  3. If a user/config file provided an invalid value for a field that is required
//     and there is no default, RETURN an error causing the app to exit.
//     - Example: DBUser.
//     - Template: err = fmt.Errorf("config: DBPath could not be validated... %w")
func (conf *File) validate() (err error) {
	//Get defaults to use for cases when user/config file field is not provided or
	//value is invalid and we can safely use a default value.
	defaults, err := newDefaultConfig()
	if err != nil {
		return
	}

	//Clean all paths, regardless of if they are used.
	conf.DBPath = filepath.Clean(filepath.ToSlash(strings.TrimSpace(conf.DBPath)))
	conf.WebFilesPath = filepath.Clean(filepath.ToSlash(strings.TrimSpace(conf.WebFilesPath)))

	//Clean results in empty paths ("") being returned as ".". This is annoying to
	//deal with; we just want blank strings if the input from the config file field
	//is blank.
	if conf.DBPath == "." {
		conf.DBPath = ""
	}
	if conf.WebFilesPath == "." {
		conf.WebFilesPath = ""
	}

	//Database related.
	conf.DBPath = filepath.FromSlash(strings.TrimSpace(conf.DBPath))
	if conf.DBPath == "" {
		conf.DBPath = defaults.DBPath
	}
	_, innerErr := os.Stat(conf.DBPath)
	if innerErr != nil && !os.IsNotExist(innerErr) {
		//Don't handle non existing database file here, it will be logged and
		//handled in main.go and database file will be created.
		return fmt.Errorf("config: DBPath could not be validated %w", innerErr)
	}

	if conf.DBJournalMode == "" {
		conf.DBJournalMode = defaults.DBJournalMode
	} else if !slices.Contains(validJournalModes, conf.DBJournalMode) {
		conf.DBJournalMode = defaults.DBJournalMode
	}

	//Web server settings.
	switch conf.WebFilesStore {
	case WebFilesStoreOnDisk:
	case WebFilesStoreEmbedded:
	default:
		conf.WebFilesStore = defaults.WebFilesStore
	}

	if conf.WebFilesStore != WebFilesStoreEmbedded {
		if conf.WebFilesPath == "" {
			conf.WebFilesPath = defaults.WebFilesPath
		}
		_, err = os.Stat(conf.WebFilesPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("config: WebFilesPath could not be validated, directory could not be found %w", err)
		} else if err != nil {
			return fmt.Errorf("config: WebFilesPath could not be validated %w", err)
		}
	}

	conf.Host = strings.TrimSpace(conf.Host)
	if conf.Host == "" {
		conf.Host = defaults.Host
	}

	if conf.Port == 0 {
		conf.Port = defaults.Port
	} else if conf.Port < portMin || conf.Port > portMax {
		conf.Port = defaults.Port
		log.Printf("WARNING! (config) Port is invalid. The value must be between %d and %d. Defaulting to %d.", portMin, portMax, conf.Port)
	}

	//User login/sessions related.
	if conf.LoginLifetimeHours <= 0 {
		conf.LoginLifetimeHours = defaults.LoginLifetimeHours
	}

	if conf.TwoFactorAuthLifetimeDays == 0 {
		conf.TwoFactorAuthLifetimeDays = defaults.TwoFactorAuthLifetimeDays
	} else if conf.TwoFactorAuthLifetimeDays < 0 {
		//Special case. If a negative number is provided, then every time a user logs
		//in they have to provide 2FA token. This may be useful for rare circumstances.
		//
		//_ = "" to remove "empty branch" staticcheck linter warning. This branch
		//is here just for the comments to explain why <0 is a special case.
		_ = ""
	}

	//Misc.
	conf.Timezone = strings.TrimSpace(conf.Timezone)
	if conf.Timezone == "" {
		conf.Timezone = defaults.Timezone
	}
	//We check if timezone provided is valid in Read() which calls
	//time.LoadLocation(conf.Timezone).

	if conf.MinPasswordLength == 0 {
		conf.MinPasswordLength = defaults.MinPasswordLength
	} else if conf.MinPasswordLength < defaults.MinPasswordLength {
		conf.MinPasswordLength = defaults.MinPasswordLength
		log.Printf("WARNING! (config) MinPasswordLength is invalid. The value must be greater than %d. Defaulting to %d.", defaults.MinPasswordLength, conf.MinPasswordLength)
	}

	if conf.PrivateKeyEncryptionKey == "" {
		log.Println("WARNING! (config) Private key encryption is disabled.")
	} else if len(conf.PrivateKeyEncryptionKey) != 32 {
		err = errors.New("config: PrivateKeyEncryptionKey must be 16, 24, or 32 characters long, 32 is recommended")
		return
	}

	//Handle development related stuff.
	if conf.Development && conf.WebFilesStore == WebFilesStoreEmbedded {
		log.Println("WARNING!")
		log.Println("WARNING! (config) Using embedded files during development, the GUI will not update as expected.")
		log.Println("WARNING!")
	}

	return
}

// print logs out the configuration file. This is used for diagnostic purposes.
// This will show all fields from the File struct, even fields that the provided
// config file omitted (except nonPublishedFields).
func (conf File) print(path string) {
	//Full path to the config file, so if file is in same directory as the
	//executable and -config flag was not provided we still get the complete path.
	pathAbs, _ := filepath.Abs(path)

	log.Println("Path to config file (-config flag):", path)
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

// Data returns the parsed config file data. This is used in other packages to use
// config file data.
func Data() File {
	return parsedConfig
}

// getRandomEncryptionKey returns a string used for encrypting the private key of a key
// pair.
func getRandomEncryptionKey() (encKey string) {
	//By default, use the longest length key per aes.NewCypher.
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
	s := sha512.Sum512([]byte(t))
	encKey = base64.StdEncoding.EncodeToString(s[:])[:l]

	return
}

// createDirectories creates the directories noted in a config file if needed.
func (conf File) createDirectories() (err error) {
	err = os.MkdirAll(filepath.Dir(conf.DBPath), 0755)
	if err != nil {
		return fmt.Errorf("config: Could not create DBPath directory. %w", err)
	}

	return
}
