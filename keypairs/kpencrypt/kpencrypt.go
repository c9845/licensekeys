/*
Package kpencrypt handles encryption of the private key of a public/private keypair.
The private key should be encrypted for security.

Encryption is symmetical and based off a password stored in the app's config file.
*/
package kpencrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"log"
)

// PasswordLength is the required length of the password, in bytes. This length is
// the maximum length supported by aes.NewCypher which is what we use to encrypt a
// private key.
const PasswordLength = 32

// PasswordLengthHex is the length of the password, when stored as a hexidecimal string.
// Each byte is represented by two hex digits.
const PasswordLengthHex = PasswordLength * 2

// NewPassword returns the password used for encryption and decryption. This
// is the password that will be saved in the app's config file.
func NewPassword() (pw string, err error) {
	//Generate new random password.
	b := make([]byte, PasswordLength)
	_, err = rand.Read(b)
	if err != nil {
		return
	}

	//Return the password as a string so it can be saved to the config file.
	pw = hex.EncodeToString(b)
	return
}

// IsCorrectLength returns true if the provided password string is the correct length
// for use in aes.NewCipher.
func IsCorrectLength(password string) bool {
	b, err := hex.DecodeString(password)
	if err != nil {
		log.Panic("kpencrypt.IsCorrectLength", "could not decode password", err)
		return false
	}

	return len(b) == PasswordLength
}

// generateKey generates a key for aes.NewCipher from password using pbkdf2.
func generateKey(password string) (key []byte, err error) {
	// salt protects against both the app's database and config file being stolen, so
	// that private keys cannot be unencrypted. However, in reality, if the db and
	// config file are stolen, the app is most likely stolen as well.
	const salt = "2btE2X08vAQ9F8pwtu8T"

	//210000 per OWASP: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#:~:text=PBKDF2%2DHMAC%2DSHA512%3A%20210%2C000%20iterations
	//32 because that is length aes.NewCipher needs.
	key, err = pbkdf2.Key(sha512.New, password, []byte(salt), 210000, 32)
	return
}

// Encrypt encrypts the provide unencryptedPrivateKey using password.
//
// This is done when a new keypair is created and before saving the keypair to the
// database, but *only* when private key encryption is enabled by a password being
// set in the config file.
//
// This uses AES-GCM encryption. The AES key is generated using PBKDF2 using password
// and a salt.
func Encrypt(unencryptedPrivateKey []byte, password string) (encryptedPrivateKey []byte, err error) {
	//Get key.
	key, err := generateKey(password)
	if err != nil {
		return
	}

	//Handle AES stuff.
	block, err := aes.NewCipher(key)
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

	//Encrypt the private key. The nonce will be included at the start of the
	//encypted value.
	encryptedPrivateKey = aesgcm.Seal(nonce, nonce, unencryptedPrivateKey, nil)
	return
}

// Decrypt decrypts the provided encryptedPrivateKey using password.
//
// This is done when a license is created for an encrypted keypair.
func Decrypt(encryptedPrivateKey []byte, password string) (unencryptedPrivateKey []byte, err error) {
	//Get key.
	key, err := generateKey(password)
	if err != nil {
		return
	}

	//Handle AES stuff.
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	//Extract the nonce. Reset the encrypted private key to not include the nonce.
	l := aesgcm.NonceSize()
	nonce, encryptedPrivateKey := encryptedPrivateKey[:l], encryptedPrivateKey[l:]

	//Decrypt.
	unencryptedPrivateKey, err = aesgcm.Open(nil, nonce, encryptedPrivateKey, nil)
	return
}
