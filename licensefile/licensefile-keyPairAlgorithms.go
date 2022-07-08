package licensefile

import (
	"fmt"

	"golang.org/x/exp/slices"
)

//KeyPairAlgoType is the key pair algorithm used to sign and verify a license key file.
type KeyPairAlgoType string

const (
	KeyPairAlgoECDSAP256 = KeyPairAlgoType("ECDSA (P256)")
	KeyPairAlgoECDSAP384 = KeyPairAlgoType("ECDSA (P384)")
	KeyPairAlgoECDSAP521 = KeyPairAlgoType("ECDSA (P521)")
	KeyPairAlgoRSA2048   = KeyPairAlgoType("RSA (2048-bit)")
	KeyPairAlgoRSA4096   = KeyPairAlgoType("RSA (4096-bit)")
	KeyPairAlgoED25519   = KeyPairAlgoType("ED25519")
)

var keyPairAlgoTypes = []KeyPairAlgoType{
	KeyPairAlgoECDSAP256,
	KeyPairAlgoECDSAP384,
	KeyPairAlgoECDSAP521,
	KeyPairAlgoRSA2048,
	KeyPairAlgoRSA4096,
	KeyPairAlgoED25519,
}

var keyPairECDSATypes = []KeyPairAlgoType{
	KeyPairAlgoECDSAP256,
	KeyPairAlgoECDSAP384,
	KeyPairAlgoECDSAP521,
}

var keyPairRSATypes = []KeyPairAlgoType{
	KeyPairAlgoRSA2048,
	KeyPairAlgoRSA4096,
}

//Valid checks if a provided algorithm is one of our supported key pair algorithms.
func (k KeyPairAlgoType) Valid() error {
	contains := slices.Contains(keyPairAlgoTypes, k)
	if contains {
		return nil
	}

	return fmt.Errorf("invalid key pair algorithm, should be one of '%s', got '%s'", keyPairAlgoTypes, k)
}
