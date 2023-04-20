package licensefile

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
)

// GenerateKeyPairED25519 creates and returns a new ED25519 private and public key.
func GenerateKeyPairED25519() (private, public []byte, err error) {
	//Generate key pair.
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return
	}

	//Encode the private key.
	x509PrivateKey, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return
	}
	pemBlockPrivateKey := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509PrivateKey,
	}
	private = pem.EncodeToMemory(pemBlockPrivateKey)

	//Encode the public key.
	x509PublicKey, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return
	}
	pemBlockPublicKey := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: x509PublicKey,
	}
	public = pem.EncodeToMemory(pemBlockPublicKey)

	return
}

// SignED25519 signs File with the provided ED25519 private key. The generated signature
// will be set in the Signature field of File. You would need to call File.Marshal()
// after this func completes to return/serve the license key file. The private key
// must be decrypted, if needed, prior to being provided.
//
// A KeyPairAlgoType is not needed since there is only one version of ED25519 that can
// be used whereas with ECDSA or RSA there are multiple versions (curve, bitsize).
func (f *File) SignED25519(privateKey []byte) (err error) {
	//Hash.
	h, err := f.hash(KeyPairAlgoED25519)
	if err != nil {
		return
	}

	//Sign the hash.
	//Decode the private key.
	pemBlock, _ := pem.Decode(privateKey)
	x509Key, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return
	}

	//Generate signature.
	sig := ed25519.Sign(x509Key.(ed25519.PrivateKey), h[:])
	if err != nil {
		return
	}

	//Encode the signature and set to the Signature field.
	err = f.encodeSignature(sig)
	if err != nil {
		return
	}

	return
}

// VerifySignatureED25519 checks if a File's signature is valid by checking it against
// the ED25519 public key. This DOES NOT check if a File is expired.
//
// This uses a copy of the File since need to remove the Signature field prior to
// hashing and verification but we don't want to modify the original File so it can
// be used as it was parsed/unmarshalled.
//
// A KeyPairAlgoType is not needed since there is only one version of ED25519 that can
// be used whereas with ECDSA or RSA there are multiple versions (curve, bitsize).
func (f File) VerifySignatureED25519(publicKey []byte) (err error) {
	//Get the decoded signature and remove the signature from the File.
	decodedSig, err := f.decodeSignature()
	if err != nil {
		return
	}
	f.Signature = ""

	//Hash.
	h, err := f.hash(KeyPairAlgoED25519)
	if err != nil {
		return
	}

	//Decode the public key.
	pemBlock, _ := pem.Decode(publicKey)
	x509Key, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return
	}

	//Verify signature.
	//Note type conversion for x509Key. ParsePKIXPublicKey returns an interface.
	valid := ed25519.Verify(x509Key.(ed25519.PublicKey), h[:], decodedSig)
	if !valid {
		err = ErrBadSignature
	}

	return
}

// VerifyED25519 checks if a File's signature is valid and if the license has expired.
// This calls VerifySignatureRSA() and Expired().
func (f File) VerifyED25519(publicKey []byte) (err error) {
	//Verify the signature.
	err = f.VerifySignatureED25519(publicKey)
	if err != nil {
		return
	}

	//Check if license is expired.
	expired, err := f.Expired()
	if err != nil {
		return
	} else if expired {
		err = ErrExpired
	}

	return
}
