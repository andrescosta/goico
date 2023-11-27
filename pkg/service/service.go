package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrescosta/goico/pkg/config"
	"github.com/andrescosta/goico/pkg/log"
	"github.com/rs/zerolog"
)

type Service func(context.Context) error

func Start(service Service) {
	StartNamed("", service)
}
func StartNamed(name string, service Service) {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	err := config.LoadEnvVariables()
	var logger *zerolog.Logger
	if name != "" {
		logger = log.NewUsingEnvAndValues(map[string]string{"service": name})
	} else {
		logger = log.NewUsingEnv()

	}

	ctx = logger.WithContext(ctx)

	if err != nil {
		logger.Fatal().Msgf("Error loading .env file: %s", err)
	}

	defer func() {
		done()
		if r := recover(); r != nil {
			logger.Fatal().Msg("error recovering")
		}
	}()

	logger.Info().Msgf("Starting process %d ", os.Getpid())
	err = service(ctx)
	done()
	if err != nil {
		logger.Fatal().Msgf("%s", err)
	}
	logger.Info().Msgf("Process %d ended ", os.Getpid())
}
