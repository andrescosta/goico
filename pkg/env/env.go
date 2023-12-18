package env

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	Development = "development"
	Production  = "production"
	Test        = "test"
)

var (
	Environment  = Development
	Environments = []string{Development, Production, Test}
)

var ErrNoEnvFileLoaded = errors.New(".env files were  not found. Configuration was not loaded")

func Env(key string, defs ...string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return getDefault(defs, "")
	}
	return s
}

func OrNil(key string) *string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	return &s
}

func AsDuration(key string, values ...time.Duration) *time.Duration {
	var def = func(v []time.Duration) *time.Duration {
		if len(v) == 0 {
			return nil
		}
		return &v[0]
	}
	s := OrNil(key)
	if s == nil {
		return def(values)
	}
	r, err := time.ParseDuration(*s)
	if err != nil {
		return def(values)
	}
	return &r
}

func AsInt[T ~int | ~int32 | ~int8 | ~int64](key string, value ...T) T {
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

func AsBool(key string, value ...bool) bool {
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

func AsArray(key string, def string) []string {
	v := Env(key, def)
	return strings.Split(v, ",")
}

// Follows this convention:
//
//	https://github.com/bkeepers/dotenv#what-other-env-files-can-i-use
func Load() error {
	loaded := false
	Environment = os.Getenv("APP_ENV")
	if strings.TrimSpace(Environment) == "" {
		Environment = Development
	} else {
		if !slices.Contains(Environments, Environment) {
			return fmt.Errorf("invalid environment %s", Environment)
		}
	}

	if err := godotenv.Load(".env." + Environment + ".local"); err == nil {
		loaded = true
	}

	if Environment != "test" {
		if err := godotenv.Load(".env.local"); err == nil {
			loaded = true
		}
	}

	if err := godotenv.Load(".env." + Environment); err == nil {
		loaded = true
	}

	if err := godotenv.Load(); err == nil {
		loaded = true
	}

	if !loaded {
		return ErrNoEnvFileLoaded
	}
	return nil
}

func getDefault[T any](values []T, default1 T) T {
	if len(values) == 0 {
		return default1
	}
	return values[0]
}
