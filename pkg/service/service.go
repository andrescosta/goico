package service

import (
	"context"
	"errors"
	"time"

	"github.com/andrescosta/goico/pkg/config"
	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/log"
	"github.com/rs/zerolog"
)

type Service struct {
	name      string
	svcType   string
	addr      *string
	startTime time.Time
	ctx       context.Context
}

var (
	ErrNotAddress = errors.New("the address was not configured")
)

func newService(ctx context.Context, name string, svcType string) *Service {
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
	addr := env.GetOrNil(name + ".addr")
	return &Service{name: name,
		addr:    addr,
		ctx:     ctx,
		svcType: svcType}
}
