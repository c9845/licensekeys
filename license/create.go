package license

import (
	"bytes"
	"context"
	"errors"
	"log"
	"time"

	"github.com/c9845/licensekeys/v4/config"
	"github.com/c9845/licensekeys/v4/db"
	"github.com/c9845/licensekeys/v4/keypairs/kpencrypt"
	"github.com/c9845/licensekeys/v4/licensefile"
	"github.com/c9845/licensekeys/v4/timestamps"
	"github.com/c9845/sqldb/v3"
	"gopkg.in/guregu/null.v3"
)

// create performs the database queries to save a new license, and generates an actual
// license file.
//
// This is used when creating a license (via GUI or API) and when renewing a license.
func create(ctx context.Context, l db.License, cfr []db.CustomFieldResult, app db.App, kp db.Keypair, renewedFromLicenseID int64) (licenseID int64, file licensefile.File, err error) {
	//Get DatetimeCreated value. This way we will have the exact same value for the
	//license data and custom field results when everything is saved to the database.
	datetimeCreated := timestamps.YMDHMS()

	l.DatetimeCreated = datetimeCreated

	//Save other data about license.
	l.IssueDate = timestamps.YMD()
	l.IssueTimestamp = time.Now().Unix()
	l.FileFormat = app.FileFormat
	l.ShowLicenseID = app.ShowLicenseID
	l.ShowAppName = app.ShowAppName

	//Start transaction since we are saving multiple things.
	c := sqldb.Connection()
	tx, err := c.BeginTxx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	//Save main license data.
	//
	//This will get us the license ID which we need to save the custom field results
	//and possible for use in the license if required per the app's details.
	//
	//The data/license will NOT be marked as Verified=true. The license file hasn't
	//been signed and verified yet so we don't want to allow it to be downloaded.
	err = l.Insert(ctx, tx)
	if err != nil {
		return
	}

	//Save custom field results.
	for _, field := range cfr {
		if l.CreatedByUserID.Valid && l.CreatedByUserID.Int64 > 0 {
			field.CreatedByUserID = null.IntFrom(l.CreatedByUserID.Int64)
		} else if l.CreatedByAPIKeyID.Valid && l.CreatedByAPIKeyID.Int64 > 0 {
			field.CreatedByAPIKeyID = null.IntFrom(l.CreatedByAPIKeyID.Int64)
		} else {
			//This should never happen!
			err = errors.New("license.create: could not determine who/what is creating the license")
			return
		}

		field.LicenseID = l.ID
		field.DatetimeCreated = datetimeCreated

		innerErr := field.Insert(ctx, tx)
		if innerErr != nil {
			return
		}
	}

	//Create the actual license file. This is done so that we can sign the license
	//data.
	file, err = buildLicense(l, cfr)
	if err != nil {
		return
	}

	//Calculate the File's fingerprint to save to the database for diagnostics. This
	//could also be used for activation in the future.
	fingerprint, err := file.CalculateFingerprint()
	if err != nil {
		return
	}

	//Sign the license.
	//
	//Decrypt the private key, if needed.
	privateKey := []byte(kp.PrivateKey)
	if kp.PrivateKeyEncrypted {
		encryptionKey := config.Data().PrivateKeyEncryptionKey

		decryptedPrivateKey, innerErr := kpencrypt.Decrypt([]byte(kp.PrivateKey), encryptionKey)
		if innerErr != nil {
			err = innerErr
			return
		}

		privateKey = decryptedPrivateKey
	}

	err = file.Sign(privateKey)
	if err != nil {
		return
	}

	//Save the fingerprint and signature to the database.
	err = l.SaveFingerprintAndSignature(ctx, tx, fingerprint, file.Signature)
	if err != nil {
		return
	}

	//Handle renewal.
	if renewedFromLicenseID > 0 {
		//Renewal relationship.
		relationship := db.RenewalRelationship{
			FromLicenseID:   renewedFromLicenseID,
			ToLicenseID:     l.ID,
			DatetimeCreated: datetimeCreated,
		}

		if l.CreatedByUserID.Valid && l.CreatedByUserID.Int64 > 0 {
			relationship.CreatedByUserID = null.IntFrom(l.CreatedByUserID.Int64)
		} else if l.CreatedByAPIKeyID.Valid && l.CreatedByAPIKeyID.Int64 > 0 {
			relationship.CreatedByAPIKeyID = null.IntFrom(l.CreatedByAPIKeyID.Int64)
		}

		err = relationship.Insert(ctx, tx)
		if err != nil {
			return
		}

		//Disable old license.
		err = db.DisableLicense(ctx, renewedFromLicenseID, tx)
		if err != nil {
			//Don't return on this error since it isn't the end of the world.
			log.Println("license.create", "renewal, could not mark 'from' license as disabled, skipping", err)
		}

		//Save renewal note.
		n := db.LicenseNote{
			LicenseID: renewedFromLicenseID,
			Note:      "License was disabled because it was renewed.",
		}
		if l.CreatedByUserID.Valid && l.CreatedByUserID.Int64 > 0 {
			n.CreatedByUserID = null.IntFrom(l.CreatedByUserID.Int64)
		} else if l.CreatedByAPIKeyID.Valid && l.CreatedByAPIKeyID.Int64 > 0 {
			n.CreatedByAPIKeyID = null.IntFrom(l.CreatedByAPIKeyID.Int64)
		}

		err = n.Insert(ctx, tx)
		if err != nil {
			return
		}
	}

	//Commit to save data.
	//
	//The license will NOT be marked as Verified=true yet. We have to verify the
	//signature we just created first. We still want to save data, however, for
	//diagnostics.
	err = tx.Commit()
	if err != nil {
		return
	}

	//Verify the just created license data and signature.
	//
	//This "writes" out the complete license file with the signature and then "reads"
	//it like a third-party app would to verify the signature with the public key.
	//This is done to confirm the signature is valid.
	err = writeReadVerify(file, []byte(kp.PublicKey))
	if err == licensefile.ErrBadSignature {
		return
	} else if err != nil {
		return
	}

	//Mark the license as verified.
	l.Verified = true
	err = l.MarkVerified(ctx)
	if err != nil {
		return
	}

	licenseID = l.ID
	return
}

// buildLicense builds File with data from the database. The File will not be signed.
//
// This is used when a creating a license so that (1) we can sign the license and
// save the signature, and (2) so that we can sign and verify the license and check
// for errors before allowing the license to be downloaded/used.
func buildLicense(l db.License, cfr []db.CustomFieldResult) (f licensefile.File, err error) {
	//Set common data in file.
	f = licensefile.File{
		CompanyName:    l.CompanyName,
		ContactName:    l.ContactName,
		PhoneNumber:    l.PhoneNumber,
		Email:          l.Email,
		IssueDate:      l.IssueDate,
		ExpirationDate: l.ExpirationDate,
	}

	//Set optional fields.
	if l.ShowLicenseID {
		f.LicenseID = l.PublicID.String()
	}
	if l.ShowAppName {
		f.AppName = l.AppName
	}

	//Add the custom field results as a map to the file.
	d := make(map[string]any, len(cfr))
	for _, f := range cfr {
		switch f.CustomFieldType {
		case db.CustomFieldTypeInteger:
			d[f.CustomFieldName] = f.IntegerValue.Int64
		case db.CustomFieldTypeDecimal:
			d[f.CustomFieldName] = f.DecimalValue.Float64
		case db.CustomFieldTypeText:
			d[f.CustomFieldName] = f.TextValue.String
		case db.CustomFieldTypeBoolean:
			d[f.CustomFieldName] = f.BoolValue.Bool
		case db.CustomFieldTypeMultiChoice:
			d[f.CustomFieldName] = f.MultiChoiceValue.String
		case db.CustomFieldTypeDate:
			d[f.CustomFieldName] = f.DateValue.String
		default:
			//This should never be hit since we validated field types when they
			//were saved/defined.
		}
	}
	f.Data = d

	return
}

// writeReadVerify is used to verify a just created license data and signature. This
// performs the same "read and verify" that a third-party app would.
func writeReadVerify(f licensefile.File, publicKey []byte) (err error) {
	//Write the license file to a buffer instead of an actual text file or writing
	//to an http response.
	b := bytes.Buffer{}
	err = f.Write(&b)
	if err != nil {
		return
	}

	//"Read" the license file.
	reread, err := licensefile.FromBytes(b.Bytes())
	if err != nil {
		return
	}

	//Verify the "reread" license.
	err = reread.Verify(publicKey)
	return
}
