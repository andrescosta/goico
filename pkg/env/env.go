package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	EnviromentVar = "APP_ENV"
	Development   = "development"
	Production    = "production"
	Test          = "test"
	WorkDirVar    = "workdir"
	BaseDirVar    = "basedir"
	fileDefault   = ".env"
)

var (
	environment  = Development
	environments = []string{Development, Production, Test}
)
var ErrNoEnvFileLoaded = errors.New(".env files were not found. Configuration was not loaded")

func String(key string, defs ...string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return getDefault(defs, "")
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
		return getDefault(value, 0)
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return getDefault(value, 0)
	}
	return T(v)
}

func Bool(key string, value ...bool) bool {
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

func Array(key string, def string) []string {
	v := String(key, def)
	return strings.Split(v, ",")
}

// Follows this convention:
//
//	https://github.com/bkeepers/dotenv#what-other-env-files-can-i-use
func Load(name string) error {
	loaded := false

	// We call it because "basedir" set
	if err := setEnvsUsingCommandLineArgs(); err != nil {
		return err
	}
	environment = os.Getenv(EnviromentVar)
	if strings.TrimSpace(environment) == "" {
		environment = Development
	} else {
		if !slices.Contains(environments, environment) {
			return fmt.Errorf("invalid environment %s", environment)
		}
	}
	if err := load(true, ".env."+environment+".local"); err == nil {
		loaded = true
	}

	if environment != "test" {
		if err := load(true, ".env.local"); err == nil {
			loaded = true
		}
	}

	if err := load(false, ".env."+environment); err == nil {
		loaded = true
	}

	if err := load(false, ".env."+name); err == nil {
		loaded = true
	}

	if err := load(false, fileDefault); err == nil {
		loaded = true
	}

	if !loaded {
		return ErrNoEnvFileLoaded
	}
	return nil
}

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

func SetargsV(name string, value string) {
	os.Args = append(os.Args, fmt.Sprintf("--env:%s=%s", name, value))
}

func Setargs(args ...string) {
	for _, arg := range args {
		os.Args = append(os.Args, fmt.Sprintf("--env:%s", arg))
	}
}

func load(override bool, files ...string) (err error) {
	for _, f := range files {
		if override {
			err = godotenv.Overload(filepath.Join(Basedir(), f))
		} else {
			err = godotenv.Load(filepath.Join(Basedir(), f))
		}
		if err != nil {
			return
		}
	}
	return
}

func getDefault[T any](values []T, default1 T) T {
	if len(values) == 0 {
		return default1
	}
	return values[0]
}
