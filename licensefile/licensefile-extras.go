package licensefile

import (
	"errors"
	"fmt"
)

// ErrFieldDoesNotExist is returned when trying to retrieve a field with a given name
// using one of the ExtraAs... type assertion retrieval functions but the field does
// not exist in the File's Extra map.
var ErrFieldDoesNotExist = errors.New("extra field does not exist")

// ExtraAsInt returns the value of the Extra field with the given name as an int64.
// If the field cannot be found, an error is returned. If the field cannot be type
// asserted to an int, an error is returned.
func (f *File) ExtraAsInt(name string) (i int, err error) {
	v, ok := f.Extras[name]
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

// ExtraAsFloat returns the value of the Extra field with the given name as a float64.
// If the field cannot be found, an error is returned. If the field cannot be type
// asserted to a float64, an error is returned.
func (f *File) ExtraAsFloat(name string) (x float64, err error) {
	v, ok := f.Extras[name]
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

// ExtraAsString returns the value of the Extra field with the given name as a string.
// If the field cannot be found, an error is returned. If the field cannot be type
// asserted to a string, an error is returned.
func (f *File) ExtraAsString(name string) (s string, err error) {
	v, ok := f.Extras[name]
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

// ExtraAsBool returns the value of the Extra field with the given name as a bool.
// If the field cannot be found, an error is returned. If the field cannot be type
// asserted to an bool, an error is returned.
func (f *File) ExtraAsBool(name string) (b bool, err error) {
	v, ok := f.Extras[name]
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
