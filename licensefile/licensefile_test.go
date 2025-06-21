package licensefile

import (
	"bytes"
	"encoding/hex"
	"os"
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
		CompanyName: "ACME Dynamite, Co.",
		ContactName: "Wyle E. Coyote",
		PhoneNumber: "123-123-1234",
		Email:       "coyote@example.com",
		Data: map[string]any{
			"roadrunner":         true,
			"sticks_of_dynamite": 100,
			"ouch":               "yes",
		},
	}
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

	fpbs := hex.EncodeToString(fpb[:])
	if fps != fpbs {
		t.Fatal("Encoding error")
		return
	}

	//Write File to a text file, and compare hashes: hash of text file vs text file's
	//contents.
}

func TestSign(t *testing.T) {
	f := File{
		CompanyName: "ACME Dynamite, Co.",
		ContactName: "Wyle E. Coyote",
		PhoneNumber: "123-123-1234",
		Email:       "coyote@example.com",
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
}

func TestVerify(t *testing.T) {
	f := File{
		CompanyName: "ACME Dynamite, Co.",
		ContactName: "Wyle E. Coyote",
		PhoneNumber: "123-123-1234",
		Email:       "coyote@example.com",
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
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		ExpireDate:  futureDate.Format("2006-01-02"),
		Data: map[string]any{
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

func TestExpiresInDays(t *testing.T) {
	//Future expiration.
	days := 10
	futureDate := time.Now().UTC().AddDate(0, 0, days)

	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		ExpireDate:  futureDate.Format("2006-01-02"),
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

	f.ExpireDate = pastDate.Format("2006-01-02")
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
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		ExpireDate:  "2006-01-02",
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
		IssueTimestamp: 1550027702,
		ExpireDate:     "2006-01-02",
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
