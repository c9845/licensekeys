package licensefile

import (
	"bytes"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateKeyPair(t *testing.T) {
	_, _, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
		return
	}
}

func TestCalculateFingerprint(t *testing.T) {
	f := File{
		LicenseID:      "01977596-fa92-76fa-aff7-ceb6ce883abb",
		AppName:        "test app",
		CompanyName:    "ACME Dynamite, Co.",
		ContactName:    "Wyle E. Coyote",
		PhoneNumber:    "123-123-1234",
		Email:          "coyote@example.com",
		IssueDate:      "1990-01-02",
		ExpirationDate: "2006-01-02",
		Data: map[string]any{
			"roadrunner":         true,
			"sticks_of_dynamite": 100,
			"ouch":               "yes",
		},
	}

	//Calculate the fingerprint as []byte and string.
	fpb, err := f.calculateFingerprint()
	if err != nil {
		t.Fatal("Could not calculate fingerprint, bytes.", err)
		return
	}

	fps, err := f.CalculateFingerprint()
	if err != nil {
		t.Fatal("Could not calculate fingerprint, string.", err)
		return
	}

	//Make sure []byte and string representations are the same.
	fpbs := hex.EncodeToString(fpb[:])
	if fps != fpbs {
		t.Fatal("Encoding error")
		return
	}

	//Set the Signature field and make sure it isn't mistakenly stripped/cleared from
	//the File.
	sig := "fake-sig-for-testing"
	f.Signature = sig
	_, err = f.CalculateFingerprint()
	if err != nil {
		t.Fatal("Could not calculate fingerprint when signature is added.", err)
		return
	}
	if f.Signature != sig {
		t.Fatal("Signature was modified, should not have been.", f.Signature)
		return
	}

	//Write File to a text file, and compare hashes: hash of text file vs text file's
	//contents.
}

func TestSign(t *testing.T) {
	f := File{
		LicenseID:      "01977596-fa92-76fa-aff7-ceb6ce883abb",
		AppName:        "test app",
		CompanyName:    "ACME Dynamite, Co.",
		ContactName:    "Wyle E. Coyote",
		PhoneNumber:    "123-123-1234",
		Email:          "coyote@example.com",
		IssueDate:      "1990-01-02",
		ExpirationDate: "2006-01-02",
		Data: map[string]any{
			"roadrunner":         true,
			"sticks_of_dynamite": 100,
			"ouch":               "yes",
		},
	}

	priv, _, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
		return
	}

	err = f.Sign(priv)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	//Test providing no private key.
	err = f.Sign([]byte(""))
	if err == nil {
		t.Fatal("Error should have occurred due to missing private key.")
		return
	}
}

func TestVerify(t *testing.T) {
	f := File{
		LicenseID:      "01977596-fa92-76fa-aff7-ceb6ce883abb",
		AppName:        "test app",
		CompanyName:    "ACME Dynamite, Co.",
		ContactName:    "Wyle E. Coyote",
		PhoneNumber:    "123-123-1234",
		Email:          "coyote@example.com",
		IssueDate:      "1990-01-02",
		ExpirationDate: "2006-01-02",
		Data: map[string]any{
			"roadrunner":         true,
			"sticks_of_dynamite": 100,
			"ouch":               "yes",
		},
	}

	priv, pub, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
		return
	}

	err = f.Sign(priv)
	if err != nil {
		t.Fatal(err)
		return
	}
	if f.Signature == "" {
		t.Fatal("Signature not saved to File.")
		return
	}

	err = f.Verify(pub)
	if err != nil {
		t.Fatal("Error with verify (see code comments!).", err)
		return
	}

	//Test with a bad public key.
	_, pubWrong, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
		return
	}
	err = f.Verify(pubWrong)
	if err == nil {
		t.Fatal("Verified with WRONG public key!! Not good!!")
		return
	}

	//Test with missing public key.
	err = f.Verify([]byte(""))
	if err == nil {
		t.Fatal("Verified with MISSING public key!! Not good!!")
		return
	}
}

func TestExpired(t *testing.T) {
	//Not expired license.
	days := 10
	futureDate := time.Now().UTC().AddDate(0, 0, days)

	f := File{
		LicenseID:      "01977596-fa92-76fa-aff7-ceb6ce883abb",
		AppName:        "test app",
		CompanyName:    "ACME Dynamite, Co.",
		ContactName:    "Wyle E. Coyote",
		PhoneNumber:    "123-123-1234",
		Email:          "coyote@example.com",
		IssueDate:      "1990-01-02",
		ExpirationDate: futureDate.Format("2006-01-02"),
		Data: map[string]any{
			"roadrunner":         true,
			"sticks_of_dynamite": 100,
			"ouch":               "yes",
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
	f.ExpirationDate = pastDate.Format("2006-01-02")

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
	f.ExpirationDate = ""
	_, err = f.Expired()
	if err != ErrMissingExpirationDate {
		t.Fatal("Error about missing expire date should have occured.")
		return
	}

	//Invalid expire date format.
	f.ExpirationDate = "01-02-2023"
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
		LicenseID:      "01977596-fa92-76fa-aff7-ceb6ce883abb",
		AppName:        "test app",
		CompanyName:    "ACME Dynamite, Co.",
		ContactName:    "Wyle E. Coyote",
		PhoneNumber:    "123-123-1234",
		Email:          "coyote@example.com",
		IssueDate:      "1990-01-02",
		ExpirationDate: futureDate.Format("2006-01-02"),
		Data: map[string]any{
			"roadrunner":         true,
			"sticks_of_dynamite": 100,
			"ouch":               "yes",
		},
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
	f.ExpirationDate = pastDate.Format("2006-01-02")
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
	f.ExpirationDate = ""
	_, err = f.ExpiresIn()
	if err != ErrMissingExpirationDate {
		t.Fatal("Error about missing expire date should have occured.")
		return
	}

	//Invalid expire date format.
	f.ExpirationDate = "01-02-2023"
	_, err = f.ExpiresIn()
	if err == nil {
		t.Fatal("Error about incorrectly formatted expire date should have occured.")
		return
	}
}

func TestExpiresInDays(t *testing.T) {
	//Future expiration.
	days := 10
	futureDate := time.Now().UTC().AddDate(0, 0, days)

	f := File{
		CompanyName:    "CompanyName",
		PhoneNumber:    "123-123-1234",
		Email:          "test@example.com",
		ExpirationDate: futureDate.Format("2006-01-02"),
	}
	daysUntilExpiration, err := f.ExpiresInDays()
	if err != nil {
		t.Fatal(err)
		return
	}
	if daysUntilExpiration <= 0 {
		t.Fatal("Days until expiration incorrect.", days, daysUntilExpiration)
		return
	}

	//Already expired.
	days = -10
	pastDate := time.Now().UTC().AddDate(0, 0, days)

	f.ExpirationDate = pastDate.Format("2006-01-02")
	daysAfterExpiration, err := f.ExpiresInDays()
	if err != nil {
		t.Fatal(err)
		return
	}
	if daysAfterExpiration >= 0 {
		t.Fatal("Days after expiration incorrect.", days, daysAfterExpiration)
		return
	}
}

func TestWriteRead(t *testing.T) {
	x, err := os.CreateTemp("", "license-key-server-test.txt")
	if err != nil {
		t.Fatal("Error creating temp file", err)
		return
	}
	defer os.Remove(x.Name())

	f := File{
		CompanyName:    "CompanyName",
		PhoneNumber:    "123-123-1234",
		Email:          "test@example.com",
		ExpirationDate: "2006-01-02",
	}

	//Write...
	err = f.Write(x)
	if err != nil {
		t.Fatal("Error writing", err)
		return
	}

	//Read...
	f2, err := FromFile(x.Name())
	if err != nil {
		t.Fatal("Error reading", err)
		return
	}

	if f2.CompanyName != f.CompanyName {
		t.Fatal("Incorrectly read written file.")
		return
	}
}

// TestLikeThirdPartyGolangApp runs through the verifying process like a Golang app
// would do.
func TestLikeThirdPartyGolangApp(t *testing.T) {
	//Create a File.
	f := File{
		LicenseID:      "01977596-fa92-76fa-aff7-ceb6ce883abb",
		AppName:        "test app",
		CompanyName:    "ACME Dynamite, Co.",
		ContactName:    "Wyle E. Coyote",
		PhoneNumber:    "123-123-1234",
		Email:          "coyote@example.com",
		IssueDate:      "1990-01-02",
		ExpirationDate: "2006-01-02",
		Data: map[string]any{
			"roadrunner":         true,
			"sticks_of_dynamite": 100,
			"ouch":               "yes",
		},
	}

	//Create keypair.
	priv, pub, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
		return
	}

	//Sign.
	err = f.Sign(priv)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Write the File to a file.
	filename := "license-key-server-test-*.txt"
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal("could not get current dir", err)
		return
	}

	x, err := os.CreateTemp(dir, filename)
	if err != nil {
		t.Fatal("Error creating temp file", err)
		return
	}
	defer os.Remove(x.Name())

	err = f.Write(x)
	if err != nil {
		t.Fatal("Error writing", err)
		return
	}
	x.Close()

	//Verify using the just-saved text file.
	t.Run("ReadFromFile", func(t *testing.T) {
		reread, err := FromFile(x.Name())
		if err != nil {
			t.Fatal("Could not read from file.", err)
			return
		}

		err = reread.Verify(pub)
		if err != nil {
			t.Fatal("Could not verify from file.", err)
			return
		}
	})

	//Verify using the just-saved text file, but read the file manually.
	t.Run("ReadFromFileManually", func(t *testing.T) {
		txtFile, err := os.ReadFile(x.Name())
		if err != nil {
			t.Fatal("Could not open file.", err)
			return
		}

		rereadFromBytes, err := FromBytes(txtFile)
		if err != nil {
			t.Fatal("Could not read from bytes.", err)
			return
		}

		err = rereadFromBytes.Verify(pub)
		if err != nil {
			t.Fatal("Could not verify from bytes.", err)
			return
		}

		rereadFromString, err := FromString(string(txtFile))
		if err != nil {
			t.Fatal("Could not read from string.", err)
			return
		}

		err = rereadFromString.Verify(pub)
		if err != nil {
			t.Fatal("Could not verify from string.", err)
			return
		}
	})

	//Verify without wring the license to an actual text file.
	t.Run("VerifyWithoutTextFile", func(t *testing.T) {
		b := bytes.Buffer{}
		err = f.Write(&b)
		if err != nil {
			t.Fatal("Could not write to buffer.", err)
			return
		}

		bb := b.Bytes()
		bbf, err := FromBytes(bb)
		if err != nil {
			t.Fatal("Could not reread from bytes.", err)
			return
		}

		err = bbf.Verify(pub)
		if err != nil {
			t.Fatal("Could not verify from bytes.", err)
			return
		}

		bs := b.String()
		bsf, err := FromString(bs)
		if err != nil {
			t.Fatal("Could not reread from string.", err)
			return
		}

		err = bsf.Verify(pub)
		if err != nil {
			t.Fatal("Could not verify from string.", err)
			return
		}
	})
}

// TestSignAndVerify runs through the signing and verifying process to make sure the
// entire process creates valid license files.
func TestSignAndVerify(t *testing.T) {
	//Filenames.
	const (
		privKey = "key--test.priv"
		pubKey  = "key--test.pub"
		licFile = "license--test.lic"
	)

	//Generate a keypair and save to file.
	privToSave, pubToSave, err := GenerateKeypair()
	if err != nil {
		log.Fatalln("Could not generate keypair.", err)
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln("Could not get working dir.", err)
		return
	}

	privFile, err := os.Create(filepath.Join(cwd, privKey))
	if err != nil {
		log.Fatalln("Could not create private key file.", err)
		return
	}
	defer privFile.Close()

	_, err = privFile.Write(privToSave)
	if err != nil {
		log.Fatalln("Could not write private key to file.", err)
		return
	}

	pubFile, err := os.Create(filepath.Join(cwd, pubKey))
	if err != nil {
		log.Fatalln("Could not create public key file.", err)
		return
	}
	defer pubFile.Close()

	_, err = pubFile.Write(pubToSave)
	if err != nil {
		log.Fatalln("Could not write public key to file.", err)
		return
	}

	//Define license file data.
	f := File{
		LicenseID:      "01977596-fa92-76fa-aff7-ceb6ce883abb",
		AppName:        "test app",
		CompanyName:    "ACME Dynamite, Co.",
		ContactName:    "Wyle E. Coyote",
		PhoneNumber:    "123-123-1234",
		Email:          "coyote@example.com",
		IssueDate:      "1990-01-02",
		ExpirationDate: "2006-01-02",
		Data: map[string]any{
			"roadrunner":         true,
			"sticks_of_dynamite": 100,
			"ouch":               "yes",
		},
	}

	//Read the private key from file.
	cwd, err = os.Getwd()
	if err != nil {
		log.Fatalln("Could not get working dir.", err)
		return
	}

	priv, err := os.ReadFile(filepath.Join(cwd, privKey))
	if err != nil {
		log.Fatalln("Could not read private key from file.", err)
		return
	}

	//Sign the license file.
	err = f.Sign(priv)
	if err != nil {
		log.Fatalln("Could not sign.", err)
		return
	}

	//Write the signed license file to an actual file.
	lic, err := os.Create(filepath.Join(cwd, licFile))
	if err != nil {
		log.Fatalln("Could not create license file.", err)
		return
	}
	defer lic.Close()

	err = f.Write(lic)
	if err != nil {
		log.Fatalln("Could not write license to file.", err)
		return
	}
	lic.Close()

	//Print out fingerprint for File, for diagnostics.
	fp, err := f.CalculateFingerprint()
	if err != nil {
		log.Fatalln("Could not calculate File fingerprint for original data.", err)
		return
	}
	log.Println("File Fingerprint, original:", fp)

	//###############################################################################
	//Read and verify.

	//Read the license file.
	rereadLic, err := FromFile(filepath.Join(cwd, licFile))
	if err != nil {
		log.Fatalln("Could not reread license file.", err)
		return
	}

	pub, err := os.ReadFile(filepath.Join(cwd, pubKey))
	if err != nil {
		log.Fatalln("Could not read public key from file.", err)
		return
	}

	//Verify.
	err = rereadLic.Verify(pub)
	if err != nil {
		log.Fatalln("Could not verify.", err)
		return
	}

	//Print out fingerprint for File, for diagnostics.
	fpReread, err := rereadLic.CalculateFingerprint()
	if err != nil {
		log.Fatalln("Could not calculate File fingerprint for reread data.", err)
		return
	}
	log.Println("File Fingerprint, reread:  ", fpReread, "match:", fp == fpReread)

}
