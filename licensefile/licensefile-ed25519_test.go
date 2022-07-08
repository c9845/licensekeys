package licensefile

import "testing"

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
		Extras: map[string]interface{}{
			"exists":   true,
			"notabool": 1,
		},
	}

	//sign
	err = f.SignED25519(private, KeyPairAlgoED25519)
	if err != nil {
		t.Fatal("Error with signing", err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not populated")
		return
	}
}

func TestVerifyED25519(t *testing.T) {
	//generate key pair
	private, public, err := GenerateKeyPairED25519()
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
	err = f.SignED25519(private, KeyPairAlgoED25519)
	if err != nil {
		t.Fatal("Error with signing", err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not populated")
		return
	}

	//verify
	err = f.VerifyED25519(public, KeyPairAlgoED25519)
	if err != nil {
		//This error gets kicked out intermittently when multiple tests are run at
		//the same time (i.e.: file-level test or package-level tests). This error
		//does not get kicked out when func-level test is run. Maybe has something
		//to do with cyrpto/rand rand.Reader when generating the key pair? Or maybe
		//with memory reuse when verifying? I really don't know...
		t.Fatal("Error with verify (see code comments!).", err)
		return
	}

	//set invalid signature and try verifying
	f.Signature = ""
	err = f.VerifyED25519(public, KeyPairAlgoED25519)
	if err != ErrBadSignature {
		t.Fatal("Error about invalid signature should have been returned.")
		return
	}
}
