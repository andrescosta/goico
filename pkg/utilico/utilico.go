package utilico

func FirstOrDefault[T any](ds []T, df T) T {
	d := df
	if len(ds) > 0 {
		d = ds[0]
	}
	return d
}

func UnwrapError(err error) []error {
	errStr := []error{err}
	e, ok := err.(interface {
		Unwrap() []error
	})
	if ok {
		errStr = e.Unwrap()
	}
	return errStr
}
