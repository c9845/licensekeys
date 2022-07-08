package licensefile

import "testing"

func TestValidKeyPairAlgos(t *testing.T) {
	//Provide a valid option.
	err := KeyPairAlgoECDSAP256.Valid()
	if err != nil {
		t.Fatal(err)
		return
	}

	//Provide an invalid option.
	f := KeyPairAlgoType("MD5")
	err = f.Valid()
	if err == nil {
		t.Fatal("error should have been returned")
		return
	}
}
