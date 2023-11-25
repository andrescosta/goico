package service

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/andrescosta/goico/pkg/config"
	"github.com/andrescosta/goico/pkg/log"
)

type Service func(context.Context) error

func Start(service Service) {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	err := config.LoadEnvVariables()

	logger := log.NewUsingEnv()

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

	err = service(ctx)
	done()

	if err != nil {
		logger.Fatal().Msgf("%s", err)
	}
}
