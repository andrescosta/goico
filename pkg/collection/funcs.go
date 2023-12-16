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
	e := make([]error, 0)
	errors.As(err, e)
	return e
	// errStr := []error{err}
	// e, ok := err.(interface {
	// 	Unwrap() []error
	// })
	// if ok {
	// 	errStr = e.Unwrap()
	// }
	// return errStr
}
