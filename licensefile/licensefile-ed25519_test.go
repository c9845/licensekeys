package licensefile

import (
	"testing"
)

func TestGenerateKeyPairED25519(t *testing.T) {
	private, public, err := GenerateKeyPairED25519()
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

func TestSignED25519(t *testing.T) {
	//generate key pair
	private, _, err := GenerateKeyPairED25519()
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
	err = f.SignED25519(private)
	if err != nil {
		t.Fatal("Error with signing", err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not populated")
		return
	}
}

func TestVerifySignatureED25519(t *testing.T) {
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
	private, public, err := GenerateKeyPairED25519()
	if err != nil {
		t.Fatal(err)
		return
	}

	//Sign.
	err = f.SignED25519(private)
	if err != nil {
		t.Fatal("Error with signing", err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not populated")
		return
	}

	//Verify.
	err = f.VerifySignatureED25519(public)
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
	err = f.VerifySignatureED25519(public)
	if err != ErrBadSignature {
		t.Fatal("Error about invalid signature should have been returned.")
		return
	}

	//Test with bad algo skipped, there is only one ED25519 algo.
}
