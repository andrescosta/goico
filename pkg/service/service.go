package service

import (
	"context"
	"time"

	"github.com/andrescosta/goico/pkg/config"
	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/log"
	"github.com/rs/zerolog"
)

type Service struct {
	Name      string
	Addr      string
	StartTime time.Time
	Ctx       context.Context
}

func NewService(ctx context.Context, name string) *Service {
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

	return &Service{Name: name,
		Addr: env.GetAsString(name + ".addr"),
		Ctx:  ctx}
}
