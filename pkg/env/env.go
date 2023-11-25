package env

import (
	"os"
	"strconv"
)

func GetAsString(key string, value ...string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return getDefault("", value)
	}
	return s
}

func GetAsInt[T ~int | ~int32 | ~int8 | ~int64](key string, value ...T) T {
	s, ok := os.LookupEnv(key)
	if !ok {
		return getDefault(0, value)
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return getDefault(0, value)
	}
	return T(v)
}

func GetAsBool(key string, value ...bool) bool {
	s, ok := os.LookupEnv(key)
	if !ok {
		return getDefault(false, value)
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return getDefault(false, value)
	}
	return v
}

func getDefault[T any](default1 T, values []T) T {
	if len(values) == 0 {
		return default1
	} else {
		return values[0]
	}
}
