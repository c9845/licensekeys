package licensefile

import (
	"encoding/base64"
	"os"
	"testing"
	"time"
)

func TestSetFileFormat(t *testing.T) {
	f := File{}
	f.SetFileFormat(FileFormatJSON)

	if f.fileFormat != FileFormatJSON {
		t.Fatal("fileFormat not set properly")
		return
	}
}

func TestFileFormat(t *testing.T) {
	f := File{}
	f.SetFileFormat(FileFormatJSON)

	if f.FileFormat() != FileFormatJSON {
		t.Fatal("fileFormat not retrieved properly")
		return
	}
}

func TestGenerateKeyPair(t *testing.T) {
	for _, k := range keyPairAlgoTypes {
		private, public, err := GenerateKeyPair(k)
		if err != nil {
			t.Fatal(err, k)
			return
		}
		if len(private) == 0 {
			t.Fatal("no private key")
			return
		}
		if len(public) == 0 {
			t.Fatal("no public key")
			return
		}
	}

	_, _, err := GenerateKeyPair(KeyPairAlgoType("bad"))
	if err == nil {
		t.Fatal("error about bad key pair algo should have been returned")
		return
	}
}

func TestSign(t *testing.T) {
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

	//Test with ecdsa key pair.
	priv, _, err := GenerateKeyPairECDSA(KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}

	err = f.Sign(priv, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	//Test with rsa key pair.
	priv, _, err = GenerateKeyPairRSA(KeyPairAlgoRSA2048)
	if err != nil {
		t.Fatal(err)
		return
	}

	err = f.Sign(priv, KeyPairAlgoRSA2048)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	//Test with ed25519 key pair.
	priv, _, err = GenerateKeyPairED25519()
	if err != nil {
		t.Fatal(err)
		return
	}

	err = f.Sign(priv, KeyPairAlgoED25519)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	//Test with bad key pair algo
	priv, _, err = GenerateKeyPairECDSA(KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}

	err = f.Sign(priv, KeyPairAlgoType("bad"))
	if err == nil {
		t.Fatal("Error about bad key pair algo should have occured.")
		return
	}
}

func TestVerifySignature(t *testing.T) {
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

	//Test with ecdsa key pair.
	priv, pub, err := GenerateKeyPairECDSA(KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}

	err = f.Sign(priv, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	err = f.VerifySignature(pub, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Test with rsa key pair.
	priv, pub, err = GenerateKeyPairRSA(KeyPairAlgoRSA2048)
	if err != nil {
		t.Fatal(err)
		return
	}

	err = f.Sign(priv, KeyPairAlgoRSA2048)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	err = f.VerifySignature(pub, KeyPairAlgoRSA2048)
	if err != nil {
		//This error gets kicked out intermittently when multiple tests are run at
		//the same time (i.e.: file-level test or package-level tests). This error
		//does not get kicked out when func-level test is run. Maybe has something
		//to do with cyrpto/rand rand.Reader when generating the key pair? Or maybe
		//with memory reuse when verifying? I really don't know...
		t.Fatal("Error with verify (see code comments!).", err)
		return
	}

	//Test with ed25519 key pair.
	priv, pub, err = GenerateKeyPairED25519()
	if err != nil {
		t.Fatal(err)
		return
	}

	err = f.Sign(priv, KeyPairAlgoED25519)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	err = f.VerifySignature(pub, KeyPairAlgoED25519)
	if err != nil {
		//This error gets kicked out intermittently when multiple tests are run at
		//the same time (i.e.: file-level test or package-level tests). This error
		//does not get kicked out when func-level test is run. Maybe has something
		//to do with cyrpto/rand rand.Reader when generating the key pair? Or maybe
		//with memory reuse when verifying? I really don't know...
		t.Fatal("Error with verify (see code comments!).", err)
		return
	}

	//test with bad signature
	f.Signature = base64.StdEncoding.EncodeToString([]byte("bad"))
	err = f.Verify(pub, KeyPairAlgoED25519)
	if err == nil {
		t.Fatal(err)
		return
	}

	//test with bad algo
	err = f.VerifySignature(pub, KeyPairAlgoType(""))
	if err == nil {
		t.Fatal("Error about bad key pair algo should have occured.")
		return
	}
}

func TestVerify(t *testing.T) {
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
		ExpireDate: time.Now().AddDate(0, 0, 10).Format("2006-01-02"),
	}

	//Test with ecdsa key pair.
	priv, pub, err := GenerateKeyPairECDSA(KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}

	err = f.Sign(priv, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	err = f.Verify(pub, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Expired...
	f.ExpireDate = time.Now().AddDate(0, 0, -10).Format("2006-01-02")

	err = f.Sign(priv, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	err = f.Verify(pub, KeyPairAlgoECDSAP256)
	if err != ErrExpired {
		t.Fatal("Error about expired license should have occured.")
		return
	}

	//Missing expiration date, should never occur.
	f.ExpireDate = ""

	err = f.Sign(priv, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	err = f.Verify(pub, KeyPairAlgoECDSAP256)
	if err != ErrMissingExpireDate {
		t.Fatal("Error about missing expire date should have occured.")
		return
	}
}

func TestHash(t *testing.T) {
	f := File{
		CompanyName: "test1",
		ContactName: "test2",
		Extras: map[string]interface{}{
			"extraString": "string",
			"extraInt":    1,
			"extraBool":   true,
		},
		fileFormat: FileFormatJSON,
	}

	for _, kp := range keyPairAlgoTypes {
		_, err := f.hash(kp)
		if err != nil {
			t.Fatal("Error during hashing", err)
			return
		}

	}

	//Test with missing file format. The file format is necessary to hash the
	//File struct to bytes that can be hashed.
	f.fileFormat = ""
	_, err := f.hash(KeyPairAlgoECDSAP256)
	if err == nil {
		t.Fatal("Error about missing file format used for hashing should have been returned")
		return
	}
}

func TestExpired(t *testing.T) {
	//Not expired license.
	days := 10
	futureDate := time.Now().UTC().AddDate(0, 0, days)

	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		fileFormat:  FileFormatJSON,
		ExpireDate:  futureDate.Format("2006-01-02"),
		Extras: map[string]interface{}{
			"exists":   true,
			"notabool": 1,
		},
	}

	expired, err := f.Expired()
	if err != nil {
		t.Fatal(err)
		return
	}
	if expired {
		t.Fatal("License is not expired.", time.Now(), futureDate)
		return
	}

	//Expired license.
	pastDate := time.Now().UTC().AddDate(0, 0, -days)
	f.ExpireDate = pastDate.Format("2006-01-02")

	expired, err = f.Expired()
	if err != nil {
		t.Fatal(err)
		return
	}
	if !expired {
		t.Fatal("License is expired, but was not noted as such")
		return
	}

	//Missing expiration date.
	f.ExpireDate = ""
	_, err = f.Expired()
	if err != ErrMissingExpireDate {
		t.Fatal("Error about missing expire date should have occured.")
		return
	}

	//Invalid expire date format.
	f.ExpireDate = "01-02-2023"
	_, err = f.Expired()
	if err == nil {
		t.Fatal("Error about incorrectly formatted expire date should have occured.")
		return
	}
}

func TestExpiresIn(t *testing.T) {
	//Future expiration.
	days := 10
	futureDate := time.Now().UTC().AddDate(0, 0, days)

	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		fileFormat:  FileFormatJSON,
		ExpireDate:  futureDate.Format("2006-01-02"),
	}
	diff, err := f.ExpiresIn()
	if err != nil {
		t.Fatal(err)
		return
	}
	if diff < 0 {
		t.Fatal("Diff should be positive for future expiration date.", diff)
		return
	}

	//Expired license.
	days = -10
	pastDate := time.Now().UTC().AddDate(0, 0, days)
	f.ExpireDate = pastDate.Format("2006-01-02")
	diff, err = f.ExpiresIn()
	if err != nil {
		t.Fatal(err)
		return
	}
	if diff > 0 {
		t.Fatal("Diff should be negative for expired license.", diff)
		return
	}

	//Missing expiration date.
	f.ExpireDate = ""
	_, err = f.ExpiresIn()
	if err != ErrMissingExpireDate {
		t.Fatal("Error about missing expire date should have occured.")
		return
	}

	//Invalid expire date format.
	f.ExpireDate = "01-02-2023"
	_, err = f.ExpiresIn()
	if err == nil {
		t.Fatal("Error about incorrectly formatted expire date should have occured.")
		return
	}
}

func TestWriteRead(t *testing.T) {
	x, err := os.CreateTemp("", "license-key-server-test.txt")
	if err != nil {
		t.Fatal("Error creating temp file", err)
		return
	}

	if err != nil {
		t.Fatal("Abs path error", err)
		return
	}
	defer os.Remove(x.Name())

	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		fileFormat:  FileFormatJSON,
		ExpireDate:  "2006-01-02",
	}

	//Write...
	err = f.Write(x)
	if err != nil {
		t.Fatal("Error writing", err)
		return
	}

	//Read...
	f2, err := Read(x.Name(), f.fileFormat)
	if err != nil {
		t.Fatal("Error reading", err)
		return
	}

	if f2.CompanyName != f.CompanyName {
		t.Fatal("Incorrectly read written file.")
		return
	}
}
