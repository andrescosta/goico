package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Workdir() string {
	defaultDir := fmt.Sprint(".", string(os.PathSeparator))
	return String(WorkDirVar, defaultDir)
}

func Basedir() string {
	defaultDir := fmt.Sprint(".", string(os.PathSeparator))
	return String(BaseDirVar, defaultDir)
}

func WorkdirPlus(elem ...string) (ret string) {
	elem = append([]string{Workdir()}, elem...)
	ret = filepath.Join(elem...)
	return
}

func String(key string, defs ...string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue(defs, "")
	}
	return s
}

func StringOrNil(key string) *string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	return &s
}

func Duration(key string, values ...time.Duration) *time.Duration {
	def := func(v []time.Duration) *time.Duration {
		if len(v) == 0 {
			return nil
		}
		return &v[0]
	}
	s := StringOrNil(key)
	if s == nil {
		return def(values)
	}
	r, err := time.ParseDuration(*s)
	if err != nil {
		return def(values)
	}
	return &r
}

func Int[T ~int | ~int32 | ~int8 | ~int64](key string, value ...T) T {
	s, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue(value, 0)
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue(value, 0)
	}
	return T(v)
}

func Bool(key string, value ...bool) bool {
	s, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue(value, false)
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return defaultValue(value, false)
	}
	return v
}

func Array(key string, def string) []string {
	v := String(key, def)
	return strings.Split(v, ",")
}

func defaultValue[T any](values []T, default1 T) T {
	if len(values) == 0 {
		return default1
	}
	return values[0]
}
