package licensefile

import (
	"testing"
)

func TestGenerateKeyPairRSA(t *testing.T) {
	for _, k := range keyPairRSATypes {
		private, public, err := GenerateKeyPairRSA(k)
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
	_, _, err := GenerateKeyPairRSA(KeyPairAlgoECDSAP256)
	if err == nil {
		t.Fatal("Error about invalid key pair algo should have been returned.")
		return
	}
}

func TestSignRSA(t *testing.T) {
	//generate key pair to use
	private, _, err := GenerateKeyPairRSA(KeyPairAlgoRSA2048)
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
		Metadata: map[string]any{
			"exists":   true,
			"notabool": 1,
		},
	}

	//sign
	err = f.SignRSA(private, KeyPairAlgoRSA2048)
	if err != nil {
		t.Fatal("Error with signing", err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not populated")
		return
	}

	//try signing with a bad key pair algo.
	err = f.SignRSA(private, KeyPairAlgoED25519)
	if err == nil {
		t.Fatal("Error about wrong key pair algo should have occured.")
		return
	}
}

func TestVerifySignatureRSA(t *testing.T) {
	//Build fake File with file format, hash type, and encoding type set.
	//Note, no expiration date. VerifySignature() doesn't do any checking
	//of the expiration date!
	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		fileFormat:  FileFormatJSON,
		Metadata: map[string]any{
			"exists":   true,
			"notabool": 1,
		},
	}

	//Generate key pair to use.
	private, public, err := GenerateKeyPairRSA(KeyPairAlgoRSA2048)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Sign.
	err = f.SignRSA(private, KeyPairAlgoRSA2048)
	if err != nil {
		t.Fatal("Error with signing", err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not populated")
		return
	}

	//Verify.
	err = f.VerifySignatureRSA(public, KeyPairAlgoRSA2048)
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
	err = f.VerifySignatureRSA(public, KeyPairAlgoRSA2048)
	if err != ErrBadSignature {
		t.Fatal("Error about invalid signature should have been returned.", err)
		return
	}

	//Test with bad algo.
	err = f.VerifySignatureRSA(public, KeyPairAlgoED25519)
	if err == nil {
		t.Fatal("Error about wrong key pair algo should have occured.")
		return
	}
}
