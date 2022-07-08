package keypairs

import (
	"bytes"
	"testing"
)

func TestEncryptPrivateKey(t *testing.T) {
	encryptionKey := "01234567890123456789012345678901"
	unencryptedData := []byte("This is a private key.")

	noncePlusEncrypted, err := encryptPrivateKey(encryptionKey, unencryptedData)
	if err != nil {
		t.Fatal(err)
		return
	}
	if len(noncePlusEncrypted) == 0 {
		t.Fatal("no encrypted data was returned, this is unexpected")
		return
	}
}

func TestDecryptPrivateKey(t *testing.T) {
	encryptionKey := "01234567890123456789012345678901"
	unencryptedData := []byte("This is a private key.")

	//have to encrypt first to test decryption
	noncePlusEncrypted, err := encryptPrivateKey(encryptionKey, unencryptedData)
	if err != nil {
		t.Fatal("Error with encryption.", err)
		return
	}

	//decrypt
	decryptedData, err := DecryptPrivateKey(encryptionKey, noncePlusEncrypted)
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
