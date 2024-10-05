package licensefile

import (
	"errors"
	"fmt"
)

// ErrFieldDoesNotExist is returned when trying to retrieve a field from the Metadata
// map of a File with a given key name using one of the MetadataAs... funcs but the
// named field does not exist in the map.
var ErrFieldDoesNotExist = errors.New("metadata field does not exist")

// MetadataAsInt returns the value of the Metadata field with the given name as an
// int. If the field cannot be found, an error is returned. If the field cannot be
// type asserted to an int, an error is returned.
func (f *File) MetadataAsInt(name string) (i int, err error) {
	v, ok := f.Metadata[name]
	if !ok {
		return 0, ErrFieldDoesNotExist
	}

	i, ok = v.(int)
	if !ok {
		err = fmt.Errorf("could not assert field to int64, field is %T", v)
		return
	}

	return
}

// MetadataAsFloat returns the value of the Metadata field with the given name as a
// float64. If the field cannot be found, an error is returned. If the field cannot be
// type asserted to a float64, an error is returned.
func (f *File) MetadataAsFloat(name string) (x float64, err error) {
	v, ok := f.Metadata[name]
	if !ok {
		return 0.0, ErrFieldDoesNotExist
	}

	x, ok = v.(float64)
	if !ok {
		err = fmt.Errorf("could not assert field to float64, field is %T", v)
		return
	}

	return
}

// MetadataAsString returns the value of the Metadata field with the given name as a
// string. If the field cannot be found, an error is returned. If the field cannot be
// type asserted to a string, an error is returned.
func (f *File) MetadataAsString(name string) (s string, err error) {
	v, ok := f.Metadata[name]
	if !ok {
		return "", ErrFieldDoesNotExist
	}

	s, ok = v.(string)
	if !ok {
		err = fmt.Errorf("could not assert field to string, field is %T", v)
		return
	}

	return
}

// MetadataAsBool returns the value of the Metadata field with the given name as a
// bool. If the field cannot be found, an error is returned. If the field cannot be
// type asserted to an bool, an error is returned.
func (f *File) MetadataAsBool(name string) (b bool, err error) {
	v, ok := f.Metadata[name]
	if !ok {
		return false, ErrFieldDoesNotExist
	}

	b, ok = v.(bool)
	if !ok {
		err = fmt.Errorf("could not assert field to bool, field is %T", v)
		return
	}

	return
}
