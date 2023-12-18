package collection

import "errors"

func FirstOrDefault[T any](ds []T, df T) T {
	d := df
	if len(ds) > 0 {
		d = ds[0]
	}
	return d
}

func UnwrapError(err error) []error {
	errorSlice := []error{err}
	var k interface {
		Unwrap() []error
	}
	if errors.As(err, &k) {
		errorSlice = k.Unwrap()
	}
	return errorSlice
}
