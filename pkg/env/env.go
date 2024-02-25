package env

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/joho/godotenv"
)

const (
	VarEnviroment = "APP_ENV"
	VarWorkDir    = "workdir"
	VarBaseDir    = "basedir"
)

const (
	Development = "development"
	Production  = "production"
	Test        = "test"
)
const fileDefault = ".env"

var environments = []string{Development, Production, Test}

// Follows this convention:
//
//	https://github.com/bkeepers/dotenv#what-other-env-files-can-i-use
func Load(name string) (bool, string, error) {
	loaded := false

	// --env:[variable]=[value] set
	if err := setEnvsUsingCommandLineArgs(); err != nil {
		return false, "", err
	}
	environment := os.Getenv(VarEnviroment)
	if strings.TrimSpace(environment) == "" {
		environment = Development
	} else {
		if !slices.Contains(environments, environment) {
			return false, "", fmt.Errorf("invalid environment %s", environment)
		}
	}
	// .env.[environment].local
	if err := load(true, ".env."+environment+".local"); err == nil {
		loaded = true
	}

	if environment != "test" {
		// .env.local
		if err := load(true, ".env.local"); err == nil {
			loaded = true
		}
	}

	//.env.local.[environment]
	if err := load(false, ".env."+environment); err == nil {
		loaded = true
	}

	//.env.[environment]
	if err := load(false, ".env."+name); err == nil {
		loaded = true
	}

	// .env
	if err := load(false, fileDefault); err == nil {
		loaded = true
	}

	return loaded, environment, nil
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
