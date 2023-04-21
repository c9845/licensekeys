package licensefile

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/exp/slices"
)

// GenerateKeyPairRSA creates and returns a new RSA private and public key. We don't
// just accept a bitsize as an input because this overall code base does not support
// every bit size.
func GenerateKeyPairRSA(k KeyPairAlgoType) (private, public []byte, err error) {
	//Make sure an RSA key pair type was provided.
	if !slices.Contains(keyPairRSATypes, k) {
		err = fmt.Errorf("invalid RSA key pair type, should be one of '%s', got '%s'", keyPairRSATypes, k)
		return
	}

	//Set correct bit size based on algo type.
	var bitSize int
	switch k {
	case KeyPairAlgoRSA2048:
		bitSize = 2048
	case KeyPairAlgoRSA4096:
		bitSize = 4096
	}

	//Generate key pair.
	pk, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return
	}

	//Encode the private key.
	x509PrivateKey := x509.MarshalPKCS1PrivateKey(pk)
	pemBlockPrivateKey := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509PrivateKey,
	}
	private = pem.EncodeToMemory(pemBlockPrivateKey)

	//Encode the public key from the private key.
	x509PublicKey := x509.MarshalPKCS1PublicKey(&pk.PublicKey)
	pemBlockPublicKey := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: x509PublicKey,
	}
	public = pem.EncodeToMemory(pemBlockPublicKey)

	return
}

// SignRSA signs File with the provided RSA private key. The generated signature will
// be set in the Signature field of File. You would need to call File.Marshal() after
// this func completes to return/serve the license key file. The private key must be
// decrypted, if needed, prior to being provided.
func (f *File) SignRSA(privateKey []byte, keyPairAlgo KeyPairAlgoType) (err error) {
	//Make sure a valid RSA algo type was provided.
	if !slices.Contains(keyPairRSATypes, keyPairAlgo) {
		err = fmt.Errorf("invalid key pair algorithm, should be one of '%s', got '%s'", keyPairRSATypes, keyPairAlgo)
		return
	}

	//Hash.
	h, err := f.hash(keyPairAlgo)
	if err != nil {
		return
	}

	//Sign the hash.
	//Decode the private key.
	pemBlock, _ := pem.Decode(privateKey)
	x509Key, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return
	}

	//Generate signature.
	sig, err := rsa.SignPSS(rand.Reader, x509Key, crypto.SHA1, h[:], nil)
	if err != nil {
		return
	}

	//Encode the signature and set to the Signature field.
	f.encodeSignature(sig)

	return
}

// VerifySignatureRSA checks if the File's signature is valid by checking it against
// the RSA public key.
//
// This DOES NOT check if a File is expired. You should call Expired() on the File
// after calling this func.
//
// This uses a copy of the File since need to remove the Signature field prior to
// hashing and verification but we don't want to modify the original File so it can
// be used as it was parsed/unmarshalled.
func (f File) VerifySignatureRSA(publicKey []byte, keyPairAlgo KeyPairAlgoType) (err error) {
	//Make sure a valid RSA algo type was provided.
	if !slices.Contains(keyPairRSATypes, keyPairAlgo) {
		err = fmt.Errorf("invalid key pair algorithm, should be one of '%s', got '%s'", keyPairRSATypes, keyPairAlgo)
		return
	}

	//Get the decoded signature and remove the signature from the File.
	decodedSig, err := f.decodeSignature()
	if err != nil {
		return
	}
	f.Signature = ""

	//Hash.
	h, err := f.hash(keyPairAlgo)
	if err != nil {
		return
	}

	//Decode the public key.
	pemBlock, _ := pem.Decode(publicKey)
	x509Key, err := x509.ParsePKCS1PublicKey(pemBlock.Bytes)
	if err != nil {
		return
	}

	//Verify signature.
	//We translate the ErrVerification to ErrBadSignature so that we can return the
	//same err as in the VerifyECDSA func.
	err = rsa.VerifyPSS(x509Key, crypto.SHA1, h[:], decodedSig, nil)
	if err == rsa.ErrVerification {
		err = ErrBadSignature
	}

	return
}

// VerifyRSA calls VerifySignatureRSA().
//
// Deprecated: This func is here just for legacy situations since the old
// VerifyRSA() func was renamed to VerifySignatureRSA() for better clarity.
// Use VerifySignatureRSA() instead.
func (f *File) VerifyRSA(publicKey []byte, keyPairAlgo KeyPairAlgoType) (err error) {
	return f.VerifySignatureRSA(publicKey, keyPairAlgo)
}
