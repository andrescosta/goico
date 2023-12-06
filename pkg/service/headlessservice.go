package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/rs/zerolog"
)

type HeadlessService struct {
	*Service
	serve       func(ctx context.Context) error
	metaService *HttpService
}

func NewHeadlessService(ctx context.Context, name string, serve func(ctx context.Context) error) (*HeadlessService, error) {
	svc := HeadlessService{}
	svc.Service = newService(ctx, name, "headless")
	svc.serve = serve
	if env.GetAsString(name+".addr", "") != "" {
		o, err := NewHttpServiceWithService(svc.Service, nil)
		if err != nil {
			return nil, err
		}
		svc.metaService = o
	}
	return &svc, nil
}

func (s HeadlessService) Start() error {
	logger := zerolog.Ctx(s.ctx)
	ctx, done := signal.NotifyContext(s.ctx, syscall.SIGINT, syscall.SIGTERM)
	s.ctx = ctx
	defer func() {
		done()
		if r := recover(); r != nil {
			logger.Fatal().Msgf("recovered from %v", r)
		}
	}()
	if s.metaService != nil {
		logger.Info().Msgf("Starting obs process %d ", os.Getpid())
		go func() {
			if err := s.metaService.Start(); err != nil {
				logger.Err(err).Msg("error obs service")
			}
		}()
	}
	logger.Info().Msgf("Starting process %d ", os.Getpid())

	s.Service.startTime = time.Now()
	err := s.serve(ctx)
	done()
	if err != nil {
		return err
	}
	return nil
}
