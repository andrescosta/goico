package env

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/joho/godotenv"
)

var Environment = Development

const (
	Development = "development"

	Production = "production"

	Test = "test"
)

var Environments = []string{Development, Production, Test}

// Follows this convention:

//

//	https://github.com/bkeepers/dotenv#what-other-env-files-can-i-use

func Populate() error {
	Environment = os.Getenv("APP_ENV")

	if strings.TrimSpace(Environment) == "" {
		Environment = Development
	} else {
		if !slices.Contains(Environments, Environment) {
			return fmt.Errorf("invalid environment %s", Environment)
		}
	}
	if err := godotenv.Load(".env." + Environment + ".local"); err != nil {
		return err
	}

	if Environment != "test" {
		if err := godotenv.Load(".env.local"); err != nil {
			return err
		}
	}

	if err := godotenv.Load(".env." + Environment); err != nil {
		return err
	}

	if err := godotenv.Load(); err != nil {
		return err
	}

	return nil
}
