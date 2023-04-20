package licensefile

import (
	"encoding/json"
	"fmt"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"
)

// FileFormat is the format of the license key file's data.
type FileFormat string

const (
	FileFormatYAML = FileFormat("yaml")
	FileFormatJSON = FileFormat("json")
)

var fileFormats = []FileFormat{
	FileFormatYAML,
	FileFormatJSON,
}

// Valid checks if a provided file format is one of our supported file formats.
func (f FileFormat) Valid() error {
	contains := slices.Contains(fileFormats, f)
	if contains {
		return nil
	}

	return fmt.Errorf("invalid file format, should be one of '%s', got '%s'", fileFormats, f)
}

// Marshal serializes a File to the format specified in the File's FileFormat.
func (f *File) Marshal() (b []byte, err error) {
	err = f.fileFormat.Valid()
	if err != nil {
		return
	}

	switch f.fileFormat {
	case FileFormatYAML:
		b, err = yaml.Marshal(f)
	case FileFormatJSON:
		//MarshalIndent is used to pretty print the encoded json, otherwise the output
		//is just on one big line which is hard for humans to read and we want to allow
		//easy inspection of the license by a human.
		b, err = json.MarshalIndent(f, "", "  ")
	}

	return
}

// Unmarshal takes data read from a file, or elsewhere, and deserializes it from the
// requested file format into a File. This is used when reading a license file for
// verifying it/the signature. If unmarshalling is successful, the format is saved to
// the File's FileFormat field. It is typically easier to call Read() instead since it
// handles reading a file from a path and deserializing it.
func Unmarshal(in []byte, format FileFormat) (f File, err error) {
	err = format.Valid()
	if err != nil {
		return
	}

	switch format {
	case FileFormatYAML:
		err = yaml.Unmarshal(in, &f)
	case FileFormatJSON:
		err = json.Unmarshal(in, &f)
	}

	//If unmarshalling was successful, save the file format to the File's FileFormat
	//field since we know the correct value. Plus, we will need the FileFormat value
	//when passed to Verify() or one of the Verify...() funcs to calculate the hash
	//correctly prior to verifying the signature.
	if err == nil {
		f.fileFormat = format
	}

	return
}

// SetFileFormat populates the fileFormat field. This func is needed since the
// fileFormat field is not exported since it is not distributed/written in a license
// file.
func (f *File) SetFileFormat(format FileFormat) {
	f.fileFormat = format
}

// FileFormat returns a File's fileFormat field. This func is needed since the
// fileFormat field is not exported since it is not distributed/writted in a license
// file.
func (f *File) FileFormat() FileFormat {
	return f.fileFormat
}
