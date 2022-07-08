package licensefile

import "testing"

func TestValidFileFormat(t *testing.T) {
	//Provide a valid option.
	err := FileFormatJSON.Valid()
	if err != nil {
		t.Fatal(err)
		return
	}

	//Provide an invalid option.
	f := FileFormat("txt")
	err = f.Valid()
	if err == nil {
		t.Fatal("error should have been returned")
		return
	}
}

func TestMarshal(t *testing.T) {
	f := File{
		CompanyName: "test1",
		ContactName: "test2",
		Extras: map[string]interface{}{
			"extraString": "string",
			"extraInt":    1,
			"extraBool":   true,
		},
	}
	for _, ff := range fileFormats {
		f.fileFormat = ff

		_, err := f.Marshal()
		if err != nil {
			t.Fatal("Marshal encountered error", err)
			return
		}
	}

	f.fileFormat = FileFormat("txt")
	_, err := f.Marshal()
	if err == nil {
		t.Fatal("Expected error about invalid file format.")
		return
	}
}

func TestUnmarshal(t *testing.T) {
	f := File{
		CompanyName: "test1",
		ContactName: "test2",
		Extras: map[string]interface{}{
			"extraString": "string",
			"extraInt":    1,
			"extraBool":   true,
		},
	}
	for _, ff := range fileFormats {
		f.fileFormat = ff

		marshalled, err := f.Marshal()
		if err != nil {
			t.Fatal("Marshal encountered error", err)
			return
		}

		//unmarshal
		out, err := Unmarshal(marshalled, ff)
		if err != nil {
			t.Fatal("Unmarshal encountered error", err)
			return
		}
		if out.CompanyName != f.CompanyName {
			t.Fatal("Unmarshal error, CompanyName mismatch")
			return
		}
		if out.Extras["extraString"] != f.Extras["extraString"] {
			t.Fatal("Unmarshal error, Extras mismatch")
			return
		}
	}

	//unknown file format
	marshalled, err := f.Marshal()
	if err != nil {
		t.Fatal("Marshal encountered error", err)
		return
	}
	_, err = Unmarshal(marshalled, FileFormat("bad"))
	if err == nil {
		t.Fatal("Error about bad file format should have occured")
		return
	}
}
