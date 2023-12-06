package env

import (
	"os"
	"strconv"
	"strings"
)

func GetAsString(key string, value ...string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return getDefault(value, "")
	}
	return s
}

func GetOrNil(key string) *string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	return &s
}

func GetAsInt[T ~int | ~int32 | ~int8 | ~int64](key string, value ...T) T {
	s, ok := os.LookupEnv(key)
	if !ok {
		return getDefault(value, 0)
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return getDefault(value, 0)
	}
	return T(v)
}

func GetAsBool(key string, value ...bool) bool {
	s, ok := os.LookupEnv(key)
	if !ok {
		return getDefault(value, false)
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return getDefault(value, false)
	}
	return v
}

func GetCommaArray(key string, def string) []string {
	v := GetAsString(key, def)
	return strings.Split(v, ",")
}

func getDefault[T any](values []T, default1 T) T {
	if len(values) == 0 {
		return default1
	} else {
		return values[0]
	}
}
