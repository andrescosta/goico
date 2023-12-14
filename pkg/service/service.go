package service

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrescosta/goico/pkg/collection"
	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/log"
	"github.com/andrescosta/goico/pkg/service/obs"
	"github.com/andrescosta/goico/pkg/service/svcmeta"
	"github.com/rs/zerolog"
)

type Service struct {
	*svcmeta.Info
	addr         *string
	startTime    time.Time
	otelProvider *obs.OtelProvider
	ctx          context.Context
	done         context.CancelFunc
}

var (
	ErrNotAddress = errors.New("the address was not configured")
	ErrOtelStack  = errors.New("error initializing otel stack")
	ErrEnvLoading = errors.New("error initializing otel stack")
)

func newService(ctx context.Context, name string, svcType string) (*Service, error) {
	// Enviroment variables configuration
	if err := env.Populate(); err != nil {
		return nil, errors.Join(err, ErrEnvLoading)
	}
	ctx, done := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)

	// Log configuration
	logger := log.NewUsingEnvAndValues(map[string]string{"service": name})
	ctx = logger.WithContext(ctx)

	envaddr := name + ".addr"
	addr := env.EnvOrNil(envaddr)
	metainfo := svcmeta.Info{Name: name, Version: "1", Type: svcType}
	o, err := obs.New(ctx, metainfo)
	if err != nil {
		return nil, errors.Join(err, ErrOtelStack)
	}

	s := &Service{
		Info:         &metainfo,
		otelProvider: o,
		addr:         addr,
		ctx:          ctx,
		done:         done,
	}

	go s.waitForDoneAndEndTheWorld()

	return s, nil
}

func (s *Service) waitForDoneAndEndTheWorld() {
	defer s.done()
	logger := zerolog.Ctx(s.ctx)
	logger.Debug().Msg("Service: waiting")
	<-s.ctx.Done()
	logger.Debug().Msg("Service closing")
	shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	err := s.otelProvider.Shutdown(shutdownCtx)
	if err != nil {
		e := collection.UnwrapError(err)
		logger.Warn().Errs(zerolog.ErrorFieldName, e).Msg("error shuting down")
	} else {
		logger.Debug().Msg("Service: stopped without errors")
	}
}
