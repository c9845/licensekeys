package licensefile

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/exp/slices"
)

//GenerateKeyPairECDSA creates and returns a new ECDSA private and public key. We
//don't just accept an elliptic.Curve as an input because this overall code base does
//not support every curve type.
func GenerateKeyPairECDSA(k KeyPairAlgoType) (private, public []byte, err error) {
	//Make sure an ECDSA key pair type was provided.
	if !slices.Contains(keyPairECDSATypes, k) {
		err = fmt.Errorf("invalid ECDSA key pair type, should be one of '%s', got '%s'", keyPairRSATypes, k)
		return
	}

	//Set correct bit size based on algo type.
	var curve elliptic.Curve
	switch k {
	case KeyPairAlgoECDSAP256:
		curve = elliptic.P256()
	case KeyPairAlgoECDSAP384:
		curve = elliptic.P384()
	case KeyPairAlgoECDSAP521:
		curve = elliptic.P521()
	}

	//Generate key pair.
	pk, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return
	}

	//Encode the private key.
	x509PrivateKey, err := x509.MarshalECPrivateKey(pk)
	if err != nil {
		return
	}
	pemBlockPrivateKey := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509PrivateKey,
	}
	private = pem.EncodeToMemory(pemBlockPrivateKey)

	//Encode the public key.
	x509PublicKey, err := x509.MarshalPKIXPublicKey(&pk.PublicKey)
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

//SignECDSA signs File with the provided ECDSA private key. The generated signature
//will be set in the Signature field of File. You would need to call File.Marshal()
//after this func completes to return/serve the license key file. The private key
//must be decrypted, if needed, prior to being provided.
func (f *File) SignECDSA(privateKey []byte, keyPairAlgo KeyPairAlgoType) (err error) {
	//Make sure a valid ECDSA algo type was provided.
	if !slices.Contains(keyPairECDSATypes, keyPairAlgo) {
		err = fmt.Errorf("invalid key pair algorithm, should be one of '%s', got '%s'", keyPairECDSATypes, keyPairAlgo)
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
	x509Key, err := x509.ParseECPrivateKey(pemBlock.Bytes)
	if err != nil {
		return
	}

	//Generate signature.
	sig, err := ecdsa.SignASN1(rand.Reader, x509Key, h[:])
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

//VerifyECDSA verifies the File's Signature with the provided ECDSA public key. You
//must populate the FileFormat field prior per to calling this func.
//
//This uses a copy of the File since we are going to remove the Signature field prior
//to hashing and verification but we don't want to modify the original File so it can
//be used as it was parsed/unmarshalled.
func (f File) VerifyECDSA(publicKey []byte, keyPairAlgo KeyPairAlgoType) (err error) {
	//Make sure a valid ECDSA algo type was provided.
	if !slices.Contains(keyPairECDSATypes, keyPairAlgo) {
		err = fmt.Errorf("invalid key pair algorithm, should be one of '%s', got '%s'", keyPairECDSATypes, keyPairAlgo)
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
	x509Key, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return
	}

	//Verify signature.
	//Note type conversion for x509Key. ParsePKIXPublicKey returns an interface.
	valid := ecdsa.VerifyASN1(x509Key.(*ecdsa.PublicKey), h[:], decodedSig)
	if !valid {
		err = ErrBadSignature
	}

	return
}
