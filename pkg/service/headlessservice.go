package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

type HeadlessService struct {
	*Service
	serve func(ctx context.Context) error
}

func NewHeadlessService(ctx context.Context, name string, serve func(ctx context.Context) error) *HeadlessService {
	svc := HeadlessService{}
	svc.Service = NewService(ctx, "worker")
	svc.serve = serve
	return &svc
}

func (s HeadlessService) Serve() error {
	logger := zerolog.Ctx(s.Ctx)
	ctx, done := signal.NotifyContext(s.Ctx, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		done()
		if r := recover(); r != nil {
			logger.Fatal().Msg("error recovering")
		}
	}()
	logger.Info().Msgf("Starting process %d ", os.Getpid())

	s.Service.StartTime = time.Now()
	err := s.serve(ctx)
	done()
	if err != nil {
		return err
	}
	return nil
}
