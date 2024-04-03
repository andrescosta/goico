package collection

import (
	"errors"
	"strings"
)

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

func JoinOf[T any](t []T, sep string, fn func(T) string) string {
	b := strings.Builder{}
	for i, v := range t {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(fn(v))
	}
	return b.String()
}
