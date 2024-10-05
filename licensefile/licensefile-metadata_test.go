package licensefile

import "testing"

func TestMetadataAsInt(t *testing.T) {
	//build fake File with file format, hash type, and encoding type set
	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		fileFormat:  FileFormatJSON,
		Metadata: map[string]any{
			"exists":   1,
			"notanint": "testing",
		},
	}

	//get field that does exist
	i, err := f.MetadataAsInt("exists")
	if err != nil {
		t.Fatal(err)
		return
	}
	if i != 1 {
		t.Fatal("Expected 1, got", i)
		return
	}

	//get field that doesn't exist
	i, err = f.MetadataAsInt("doesnotexist")
	if err != ErrFieldDoesNotExist {
		t.Fatal("Error about non-existant field should have occured.")
		return
	}
	if i != 0 {
		t.Fatal("Expected 0, got", i)
		return
	}

	//get not an int field
	i, err = f.MetadataAsInt("notanint")
	if err == nil {
		t.Fatal("Expected type assertion error.")
		return
	}
	if i != 0 {
		t.Fatal("Expected 0, got", i)
		return
	}
}

func TestMetadataAsFloat(t *testing.T) {
	//build fake File with file format, hash type, and encoding type set
	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		fileFormat:  FileFormatJSON,
		Metadata: map[string]any{
			"exists":    1.234,
			"notafloat": "testing",
		},
	}

	//get field that does exist
	x, err := f.MetadataAsFloat("exists")
	if err != nil {
		t.Fatal(err)
		return
	}
	if x != 1.234 {
		t.Fatal("Expected 1.234, got", x)
		return
	}

	//get field that doesn't exist
	x, err = f.MetadataAsFloat("doesnotexist")
	if err != ErrFieldDoesNotExist {
		t.Fatal("Error about non-existant field should have occured.")
		return
	}
	if x != 0 {
		t.Fatal("Expected 0, got", x)
		return
	}

	//get not a float field
	x, err = f.MetadataAsFloat("notafloat")
	if err == nil {
		t.Fatal("Expected type assertion error.")
		return
	}
	if x != 0 {
		t.Fatal("Expected 0, got", x)
		return
	}
}

func TestMetadataAsString(t *testing.T) {
	//build fake File with file format, hash type, and encoding type set
	f := File{
		CompanyName: "CompanyName",
		PhoneNumber: "123-123-1234",
		Email:       "test@example.com",
		fileFormat:  FileFormatJSON,
		Metadata: map[string]any{
			"exists":     "hello-world",
			"notastring": 1,
		},
	}

	//get field that does exist
	s, err := f.MetadataAsString("exists")
	if err != nil {
		t.Fatal(err)
		return
	}
	if s != "hello-world" {
		t.Fatal("Expected hello-world, got", s)
		return
	}

	//get field that doesn't exist
	s, err = f.MetadataAsString("doesnotexist")
	if err != ErrFieldDoesNotExist {
		t.Fatal("Error about non-existant field should have occured.")
		return
	}
	if s != "" {
		t.Fatal("Expected '', got", s)
		return
	}

	//get not a string field
	s, err = f.MetadataAsString("notastring")
	if err == nil {
		t.Fatal("Expected type assertion error.")
		return
	}
	if s != "" {
		t.Fatal("Expected '', got", s)
		return
	}
}

func TestMetadataAsBool(t *testing.T) {
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

	//get field that does exist
	b, err := f.MetadataAsBool("exists")
	if err != nil {
		t.Fatal(err)
		return
	}
	if b != true {
		t.Fatal("Expected 'true', got", b)
		return
	}

	//get field that doesn't exist
	b, err = f.MetadataAsBool("doesnotexist")
	if err != ErrFieldDoesNotExist {
		t.Fatal("Error about non-existant field should have occured.")
		return
	}
	if b != false {
		t.Fatal("Expected '', got", b)
		return
	}

	//get not a string field
	b, err = f.MetadataAsBool("notabool")
	if err == nil {
		t.Fatal("Expected type assertion error.")
		return
	}
	if b != false {
		t.Fatal("Expected '', got", b)
		return
	}
}
