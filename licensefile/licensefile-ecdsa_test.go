package licensefile

import (
	"testing"
)

func TestGenerateKeyPairECDSA(t *testing.T) {
	for _, k := range keyPairECDSATypes {
		private, public, err := GenerateKeyPairECDSA(k)
		if err != nil {
			t.Fatal(err)
			return
		}
		if len(private) == 0 {
			t.Fatal("No private key returned.")
			return
		}
		if len(public) == 0 {
			t.Fatal("No public key returned.")
			return
		}
	}

	//try with invalid algo
	_, _, err := GenerateKeyPairECDSA(KeyPairAlgoRSA2048)
	if err == nil {
		t.Fatal("Error about invalid key pair algo should have been returned.")
		return
	}
}

func TestSignECDSA(t *testing.T) {
	//generate key pair to use.
	private, _, err := GenerateKeyPairECDSA(KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}

	//build fake File with file format, hash type, and encoding type set
	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		fileFormat:  FileFormatJSON,
		Extras: map[string]interface{}{
			"exists":   true,
			"notabool": 1,
		},
	}

	//sign
	err = f.SignECDSA(private, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal("Error with signing", err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not populated")
		return
	}

	//try signing with a bad key pair algo.
	err = f.SignECDSA(private, KeyPairAlgoRSA2048)
	if err == nil {
		t.Fatal("Error about wrong key pair algo should have occured.")
		return
	}
}

func TestVerifySignatureECDSA(t *testing.T) {
	//Build fake File with file format, hash type, and encoding type set.
	//Note, no expiration date. VerifySignature() doesn't do any checking
	//of the expiration date!
	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		fileFormat:  FileFormatJSON,
		Extras: map[string]interface{}{
			"exists":   true,
			"notabool": 1,
		},
	}

	//Generate key pair to use.
	private, public, err := GenerateKeyPairECDSA(KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Sign.
	err = f.SignECDSA(private, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal("Error with signing", err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not populated")
		return
	}

	//Verify.
	err = f.VerifySignatureECDSA(public, KeyPairAlgoECDSAP256)
	if err != nil {
		//This error gets kicked out intermittently when multiple tests are run at
		//the same time (i.e.: file-level test or package-level tests). This error
		//does not get kicked out when func-level test is run. Maybe has something
		//to do with cyrpto/rand rand.Reader when generating the key pair? Or maybe
		//with memory reuse when verifying? I really don't know...
		t.Fatal("Error with verify (see code comments!).", err)
		return
	}

	//Test with bad signature.
	f.Signature = ""
	err = f.VerifySignatureECDSA(public, KeyPairAlgoECDSAP256)
	if err != ErrBadSignature {
		t.Fatal("Error about invalid signature should have been returned.")
		return
	}

	//Test with bad algo.
	err = f.VerifySignatureECDSA(public, KeyPairAlgoRSA2048)
	if err == nil {
		t.Fatal("Error about wrong key pair algo should have occured.")
		return
	}

}
