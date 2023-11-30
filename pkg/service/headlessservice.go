package service

import (
	"context"
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
	logger := zerolog.Ctx(s.ctx)
	ctx, done := signal.NotifyContext(s.ctx, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		done()
		if r := recover(); r != nil {
			logger.Fatal().Msg("error recovering")
		}
	}()
	s.Service.StartTime = time.Now()
	err := s.serve(ctx)
	done()
	if err != nil {
		return err
	}
	return nil
}
