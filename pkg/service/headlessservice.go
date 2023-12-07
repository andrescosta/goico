package service

import (
	"context"
	"os"
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
	s, err := newService(ctx, name, "headless")
	if err != nil {
		return nil, err
	}
	svc := HeadlessService{
		Service: s,
		serve:   serve,
	}
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
	err := s.serve(s.ctx)
	if err != nil {
		return err
	}
	return nil
}
