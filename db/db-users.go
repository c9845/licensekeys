package db

//This table stores the users for this app.

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/c9845/licensekeys/v2/timestamps"
	"github.com/c9845/licensekeys/v2/users/pwds"
	"github.com/c9845/sqldb/v3"
	"github.com/jmoiron/sqlx"
)

// TableUsers is the name of the table
const TableUsers = "users"

// InitialUserUsername is the username of the default user that is created when
// --deploy-db is run for the first time.
const InitialUserUsername = "admin@example.com"

// InitialUserPassword is populated by insertInitialUser() for the default user when
// the database is deployed. This value is then logged out when --deploy-db is done
// running.
//
// This was implemented so that we can have a random password upon each deploy instead
// of having a hardcoded default password. This forces users to not use the default
// user.
var InitialUserPassword = ""

// User is used to interact with the table
type User struct {
	//Basics.
	ID               int64
	DatetimeCreated  string
	DatetimeModified string
	CreatedByUserID  int64
	Active           bool

	Username            string
	Password            string `json:"-"`
	BadPasswordAttempts uint8  `json:"-"`

	//Access control permissions.
	Administrator  bool //can user create other users, apps, signing details, etc.
	CreateLicenses bool //can user create licenses.
	ViewLicenses   bool //can user view and download licenses.

	//2 factor auth stuff
	//Using "Two" not "2" since golang struct fields must start with letter.
	TwoFactorAuthEnabled     bool   //does the user use 2 factor auth
	TwoFactorAuthSecret      string `json:"-"` //the shared secret used to validate 2fa tokens
	TwoFactorAuthBadAttempts uint8  `json:"-"` //the number of bad 2fa tokens provides, increases the time taken to verify tokens to reduce impact of brute forcing 2fa tokens

	//Have to use different field names since struct tags block using Password field.
	//Only used when sending data into app; adding user or updating password.
	PasswordInput1 string
	PasswordInput2 string

	//JOINed fields
}

const (
	createTableUsers = `
		CREATE TABLE IF NOT EXISTS ` + TableUsers + `(
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			DatetimeModified TEXT DEFAULT CURRENT_TIMESTAMP,
			CreatedByUserID INTEGER NOT NULL,
			Active INTEGER NOT NULL DEFAULT 1,
			
			Username TEXT NOT NULL,
			Password TEXT NOT NULL,
			BadPasswordAttempts INTEGER NOT NULL DEFAULT 0,

			Administrator INTEGER NOT NULL DEFAULT 0,
			CreateLicenses INTEGER NOT NULL DEFAULT 0,
			ViewLicenses INTEGER NOT NULL DEFAULT 0,
			
			TwoFactorAuthEnabled INTEGER NOT NULL DEFAULT 0,
			TwoFactorAuthSecret TEXT NOT NULL DEFAULT '',
			TwoFactorAuthBadAttempts INTEGER NOT NULL DEFAULT 0
		)
	`

	createIndexUsersUsername = `CREATE INDEX IF NOT EXISTS ` + TableUsers + `__Username_idx ON ` + TableUsers + ` (Username)`
	createIndexUsersActive   = `CREATE INDEX IF NOT EXISTS ` + TableUsers + `__Active_idx ON ` + TableUsers + ` (Active)`
)

func insertInitialUser(c *sqlx.DB) (err error) {
	//Check if the default initial user already exists.
	ctx := context.Background()
	_, err = GetUserByUsername(ctx, InitialUserUsername, sqldb.Columns{"ID"})
	if err == nil {
		log.Println("insertInitialUser...skipping, default initial user already exists")
		return
	} else if err != sql.ErrNoRows {
		return
	}

	//Default user doesn't exist. Check if other users exist. This handles if the
	//default initial user's username was changed so we don't recreate the default
	//user for no reason.
	uu, err := GetUsers(ctx, true)
	if err != nil {
		return
	} else if len(uu) > 0 {
		log.Println("insertInitialUser...skipping, other users already exist")
		return
	}

	//No users exist in the database, as expected for an initial deploy of the app.
	//Create the default initial user.
	b := make([]byte, 21)
	_, err = rand.Read(b)
	if err == nil {
		InitialUserPassword = base64.StdEncoding.EncodeToString(b)

	} else {
		log.Println("insertInitialUser...failed creating random password for default initial user, falling back to less-random password", err)

		now := time.Now().UnixNano()
		InitialUserPassword = strconv.FormatInt(now, 10)
	}

	hashedPwd, err := pwds.Create(InitialUserPassword)
	if err != nil {
		return
	}

	u := User{
		Active:          true,
		Administrator:   true,
		CreateLicenses:  true,
		ViewLicenses:    true,
		CreatedByUserID: 0, //since this user is the initial user, no one created it
		Username:        InitialUserUsername,
		Password:        hashedPwd,
	}

	err = u.Insert(ctx)
	return
}

// GetUsers looks up a list of users.
func GetUsers(ctx context.Context, activeOnly bool) (uu []User, err error) {
	q := `
		SELECT *
		FROM ` + TableUsers + ` 
	`

	b := sqldb.Bindvars{}
	if activeOnly {
		q += `WHERE (Active = ?)`
		b = append(b, true)
	}

	q += ` ORDER BY Active DESC, Username ASC`

	c := sqldb.Connection()
	err = c.SelectContext(ctx, &uu, q, b...)
	return
}

// GetUserByID looks up a user's data by their ID.
func GetUserByID(ctx context.Context, id int64, columns sqldb.Columns) (u User, err error) {
	cols, err := columns.ForSelect()
	if err != nil {
		return
	}

	q := `
		SELECT ` + cols + `
		FROM ` + TableUsers + `
		WHERE ID=?
	`

	c := sqldb.Connection()
	err = c.GetContext(ctx, &u, q, id)
	return
}

// GetUserByUsername looks up a user's by their username.
func GetUserByUsername(ctx context.Context, username string, columns sqldb.Columns) (u User, err error) {
	cols, err := columns.ForSelect()
	if err != nil {
		return
	}

	q := `
		SELECT ` + cols + `
		FROM ` + TableUsers + `
		WHERE Username = ?
	`

	c := sqldb.Connection()
	err = c.GetContext(ctx, &u, q, username)
	return
}

// Validate handles validating and sanitizing of data prior to saving or updating
// a user.
func (u *User) Validate(ctx context.Context) (errMsg string, err error) {
	//Santize.
	u.Username = strings.ToLower(strings.TrimSpace(u.Username))

	//Validate.
	if u.Username == "" {
		errMsg = "You must provide the user's username.  This must be an email address."
		return
	}

	//Make sure username is actually an email address.
	//This differs from the regex in common.ts since that regex didn't work in go for
	//some odd reason. We cannot use backtick encapsulated string b/c the backtick
	//symbol is used in the regex pattern.
	rx, err := regexp.Compile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	if err != nil {
		log.Println("Could not compile regex to validate username.", err)
	} else {
		if !rx.MatchString(u.Username) {
			errMsg = "The username you provided is not a valid email address."
			return
		}
	}

	//Check if a user with this username already exists. This uses the ID value to
	//handle if we are updating an already existing user where the same name would
	//be allowed. We don't care if a user is inactive.
	existing, err := GetUserByUsername(ctx, u.Username, sqldb.Columns{"ID"})
	if err != nil && err != sql.ErrNoRows {
		return "Could not look up if a user with this username already exists.", err
	} else if (err == nil && u.ID > 0 && u.ID != existing.ID) || (err == nil && u.ID == 0) {
		errMsg = "A user with this username already exists."
		return
	}

	//Make sure any related permissions are set properly. Ex.: If a user has a "Write"
	//permission, make sure the related "Read" permission is also assigned.
	//
	//This helps prevent issues with "permission chains" where multiple permissions
	//are needed to complete certain tasks (ex.: receiving also needs suppliers read
	//and raw materials read permissions to sort by what raw material is being
	//received).
	//
	//This matches code in users.ts.
	if u.CreateLicenses {
		u.ViewLicenses = true
	}

	return "", nil
}

// Insert saves a user to the database.
func (u *User) Insert(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"Username",
		"Password",
		"Active",
		"CreatedByUserID",
		"Administrator",
		"CreateLicenses",
		"ViewLicenses",
	}
	b := sqldb.Bindvars{
		u.Username,
		u.Password,
		u.Active,
		u.CreatedByUserID,
		u.Administrator,
		u.CreateLicenses,
		u.ViewLicenses,
	}
	colString, valString, err := cols.ForInsert()
	if err != nil {
		return
	}

	q := `INSERT INTO ` + TableUsers + `(` + colString + `) VALUES (` + valString + `)`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, b...)
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	u.ID = id
	return
}

// Update saves changes to a user
func (u *User) Update(ctx context.Context) (err error) {
	cols := sqldb.Columns{
		"Username",
		"Active",
		"Administrator",
		"CreateLicenses",
		"ViewLicenses",
	}
	colString, err := cols.ForUpdate()
	if err != nil {
		return
	}

	q := `UPDATE ` + TableUsers + ` SET ` + colString + ` WHERE ID = ?`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		u.Username,
		u.Active,
		u.Administrator,
		u.CreateLicenses,
		u.ViewLicenses,

		u.ID,
	)
	return
}

// SetNewPassword sets a new password for a given user ID. The password should
// already be hashed.
func SetNewPassword(ctx context.Context, userID int64, passwordHash string) (err error) {
	q := `
		UPDATE ` + TableUsers + `
		SET 
			Password = ?
		WHERE 
			ID = ?
	`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		passwordHash,

		userID,
	)
	return
}

// Save2FASecret saves the secret shared secret for 2fa to the database for a user.
// This does not enable 2fa since the user still needs to verify a 2fa token the first
// time a secret/qr code is shown to them.
func Save2FASecret(ctx context.Context, userID int64, secret string) (err error) {
	q := `
		UPDATE ` + TableUsers + `
		SET TwoFactorAuthSecret = ?
		WHERE ID = ?
	`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		secret,

		userID,
	)
	return
}

// Enable2FAForAll is a random value that allows the use of the Enable2FA func to
// turn 2FA on/off for all users. This random value is used so that someone can't
// just code in a value easily when using this func and mistakenly turn 2FA on/off
// for all users.
const Enable2FAForAll int64 = -132674

// Enable2FA sets 2fa on or off for a user.
//
// This also works for all users, but only if userID is set to Enable2FAForAll. Be
// careful! This was added to support turning 2fa on/off in development and testing.
func Enable2FA(ctx context.Context, userID int64, turnOn bool) (err error) {
	cols := sqldb.Columns{
		"DatetimeModified",
		"TwoFactorAuthEnabled",
		"TwoFactorAuthBadAttempts",
	}
	b := sqldb.Bindvars{
		timestamps.YMDHMS(),
		turnOn,
		0,
	}
	colString, err := cols.ForUpdate()
	if err != nil {
		return
	}

	q := `UPDATE ` + TableUsers + ` SET ` + colString + ` WHERE ID = ?`

	if userID == Enable2FAForAll {
		q = strings.Replace(q, "WHERE ID = ?", "WHERE ID > 0", 1)
	} else {
		b = append(b, userID)
	}

	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, b...)
	return
}

// Set2FABadAttempts sets the value for 2fa bad login attempts for a user. This is
// used to either (a) reset the value upon a good login or (b) increment the value
// (up to the max) for bad logins. The new badValue should have already been
// calculated and validated.
func Set2FABadAttempts(ctx context.Context, userID int64, badValue uint8) error {
	q := `
		UPDATE ` + TableUsers + `
		SET TwoFactorAuthBadAttempts = ?
		WHERE ID = ?
	`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		badValue,

		userID,
	)
	return err
}

// SetPasswordBadAttempts sets the value for the bad password attempts for a user.
// This is used to either (a) reset the value upon a good password or (b) increment
// the value (up to the max) for bad passwords. The new badValue should have already
// been calculated and validated.
func SetPasswordBadAttempts(ctx context.Context, userID int64, badValue uint8) error {
	q := `
		UPDATE ` + TableUsers + `
		SET BadPasswordAttempts = ?
		WHERE ID = ?
	`
	c := sqldb.Connection()
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,

		badValue,

		userID,
	)
	return err
}
