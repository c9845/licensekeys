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

	err = f.Verify(pub, KeyPairAlgoRSA2048)
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

	err = f.Verify(pub, KeyPairAlgoED25519)
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
	err = f.Verify(pub, KeyPairAlgoType(""))
	if err == nil {
		t.Fatal("Error about bad key pair algo should have occured.")
		return
	}
}

func TestDaysUntilExpired(t *testing.T) {
	days := 10
	futureDate := time.Now().UTC().AddDate(0, 0, days)

	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		fileFormat:  FileFormatJSON,
		ExpireDate:  futureDate.Format("2006-01-02"),
	}

	diff, err := f.DaysUntilExpired()
	if err != nil {
		t.Fatal(err)
		return
	}
	if diff != days {
		t.Fatalf("Date diff mismatch, expected %d, got %d", days, diff)
		return
	}

	//handle missing expire date
	f.ExpireDate = ""
	diff, err = f.DaysUntilExpired()
	if err == nil {
		t.Fatal("Error about missing expire date was expected.")
		return
	}
	if diff != 0 {
		t.Fatal("Date diff should be 0 but was", diff)
		return
	}

	//handle date in the past aka an expired license
	days = -10
	f.ExpireDate = time.Now().AddDate(0, 0, days).Format("2006-01-02")
	diff, err = f.DaysUntilExpired()
	if err != nil {
		t.Fatal(err)
		return
	}
	if diff != days {
		t.Fatalf("Date diff mismatch, expected %d, got %d", days, diff)
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

func TestReverifyEvery(t *testing.T) {
	//build fake File with file format, hash type, and encoding type set
	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		Extras: map[string]interface{}{
			"exists":   true,
			"notabool": 1,
		},
	}
	f.fileFormat = FileFormatYAML

	//Create keypair for signing.
	priv, pub, err := GenerateKeyPairECDSA(KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Sign.
	err = f.Sign(priv, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	//Write to temp file since reverifiying needs to read from a file.
	temp, err := os.CreateTemp("", "temp-test-license.txt")
	if err != nil {
		t.Fatal(err)
		return
	}
	defer temp.Close()

	err = f.Write(temp)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Read and Verify.
	readFile, err := Read(temp.Name(), f.fileFormat)
	if err != nil {
		t.Fatal(err)
		return
	}

	err = readFile.Verify(pub, KeyPairAlgoECDSAP256)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Reverify.
	err = readFile.Reverify()
	if err != nil {
		t.Fatal(err)
		return
	}

	//Edit something in the license file on disk and try reverifying.
	rereadFile, err := Read(temp.Name(), f.fileFormat)
	if err != nil {
		t.Fatal(err)
		return
	}
	rereadFile.CompanyName = ""

	err = rereadFile.Write(temp)
	if err != nil {
		t.Fatal(err)
		return
	}

	err = readFile.Reverify()
	if err == nil {
		t.Fatal("signature should be invalid during reverification")
		return
	}

}
