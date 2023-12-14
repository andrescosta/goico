package env

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func Env(key string, defs ...string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return getDefault(defs, "")
	}
	return s
}

func EnvOrNil(key string) *string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	return &s
}

func EnvAsDuration(key string, values ...time.Duration) *time.Duration {
	var def = func(v []time.Duration) *time.Duration {
		if len(v) == 0 {
			return nil
		} else {
			return &v[0]
		}
	}
	s := EnvOrNil(key)
	if s == nil {
		return def(values)
	}
	r, err := time.ParseDuration(*s)
	if err != nil {
		return def(values)
	}
	return &r
}

func EnvAsInt[T ~int | ~int32 | ~int8 | ~int64](key string, value ...T) T {
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

func EnvAsBool(key string, value ...bool) bool {
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

func EnvAsArray(key string, def string) []string {
	v := Env(key, def)
	return strings.Split(v, ",")
}

func getDefault[T any](values []T, default1 T) T {
	if len(values) == 0 {
		return default1
	} else {
		return values[0]
	}
}
