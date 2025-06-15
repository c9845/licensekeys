package kpencrypt

import (
	"bytes"
	"testing"
)

func TestNewPassword(t *testing.T) {
	pw, err := NewPassword()
	if err != nil {
		t.Errorf("Error generating new password: %v", err)
	}

	if len(pw) != PasswordLength {
		t.Errorf("Password length is incorrect. Expected %d, got %d", PasswordLength, len(pw))
	}
}

func TestEncrypt(t *testing.T) {
	pw, err := NewPassword()
	if err != nil {
		t.Fatal(err)
		return
	}

	unencryptedData := []byte("This is a private key.")

	encryptedData, err := Encrypt(unencryptedData, pw)
	if err != nil {
		t.Fatal(err)
		return
	}

	if len(encryptedData) == 0 {
		t.Fatal("no encrypted data was returned, this is unexpected")
		return
	}
}

func TestDecryptPrivateKey(t *testing.T) {
	pw, err := NewPassword()
	if err != nil {
		t.Fatal(err)
		return
	}

	unencryptedData := []byte("This is a private key.")

	//have to encrypt first to test decryption
	encryptedData, err := Encrypt(unencryptedData, pw)
	if err != nil {
		t.Fatal(err)
		return
	}

	//decrypt
	decryptedData, err := Decrypt(encryptedData, pw)
	if err != nil {
		t.Fatal("Error with decryption.", err)
		return
	}
	if !bytes.Equal(decryptedData, unencryptedData) {
		t.Fatal("mismatch via byte compare")
		return
	}
	if string(decryptedData) != string(unencryptedData) {
		t.Fatal("mismatch via string compare")
		return
	}
}
