package licensefile

import (
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

func TestSign(t *testing.T) {
	//build fake File with file format, hash type, and encoding type set
	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		Data: map[string]any{
			"exists":   true,
			"notabool": 1,
		},
	}

	//Test with ed25519 key pair.
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
	//Build fake File with file format, hash type, and encoding type set.
	//Note, no expiration date. VerifySignature() doesn't do any checking
	//of the expiration date!
	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		Data: map[string]any{
			"exists":   true,
			"notabool": 1,
		},
	}

	//Test with ED25519 key pair.
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
}

func TestFingerprint(t *testing.T) {
	f := File{
		CompanyName: "test1",
		ContactName: "test2",
		Data: map[string]any{
			"extraString": "string",
			"extraInt":    1,
			"extraBool":   true,
		},
	}

	_, err := f.fingerprint()
	if err != nil {
		t.Fatal("Error during hashing", err)
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
	f2, err := Read(x.Name())
	if err != nil {
		t.Fatal("Error reading", err)
		return
	}

	if f2.CompanyName != f.CompanyName {
		t.Fatal("Incorrectly read written file.")
		return
	}
}
