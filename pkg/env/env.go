package env

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
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
	envWorkDir  = "workdir"
	envBaseDir  = "basedir"
	fileDefault = ".env"
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
func Load(name string) error {
	loaded := false

	// We call it because "basedir" set
	setEnvsUsingCommandLineArgs()
	Environment = os.Getenv("APP_ENV")
	if strings.TrimSpace(Environment) == "" {
		Environment = Development
	} else {
		if !slices.Contains(Environments, Environment) {
			return fmt.Errorf("invalid environment %s", Environment)
		}
	}

	if err := loadUsingGoDot(".env." + Environment + ".local"); err == nil {
		loaded = true
	}

	if Environment != "test" {
		if err := loadUsingGoDot(".env.local"); err == nil {
			loaded = true
		}
	}

	if err := loadUsingGoDot(".env." + Environment); err == nil {
		loaded = true
	}

	if err := loadUsingGoDot(".env." + name); err == nil {
		loaded = true
	}

	if err := loadUsingGoDot(fileDefault); err == nil {
		loaded = true
	}

	if !loaded {
		return ErrNoEnvFileLoaded
	}
	// We call it again to override the env values with command line ones
	setEnvsUsingCommandLineArgs()
	return nil
}

func getDefault[T any](values []T, default1 T) T {
	if len(values) == 0 {
		return default1
	}
	return values[0]
}

func WorkDir() string {
	return Env(envWorkDir, fmt.Sprint(".", string(os.PathSeparator)))
}

func InWorkDir(dir ...string) (ret string) {
	dir = append([]string{WorkDir()}, dir...)
	ret = path.Join(dir...)
	return
}

func BaseDir() string {
	return Env(envBaseDir, fmt.Sprint(".", string(os.PathSeparator)))
}

func getDir(dir string) string {
	return filepath.Join(BaseDir(), dir)
}

func loadUsingGoDot(files ...string) (err error) {
	for _, f := range files {
		err = godotenv.Load(getDir(f))
		if err != nil {
			return
		}
	}
	return
}
