package process

import (
	"context"
	"os"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/http"
	"github.com/rs/zerolog"
)

type TypeFn func(ctx context.Context) error

type Service struct {
	service       *service.Service
	process       TypeFn
	helperService *http.Service
}

type Option struct {
	serveHandler  TypeFn
	healthCheckFN http.HealthChkFn
	ctx           context.Context
	name          string
	addr          string
}

func New(opts ...func(*Option)) (*Service, error) {
	opt := &Option{}
	for _, o := range opts {
		o(opt)
	}
	s := &Service{}
	service, err := service.New(
		service.WithName(opt.name),
		service.WithContext(opt.ctx),
		service.WithKind("headless"),
	)
	if err != nil {
		return nil, err
	}
	s.service = service
	s.process = opt.serveHandler
	addr := opt.addr
	if addr == "" {
		addr = env.String(opt.name+".addr", "")
	}
	if addr != "" {
		// creates an HTTP service to serve metadata and health information
		// about this headless(no rest or grpc interface) service.
		h, err := http.NewWithServiceContainer(
			http.WithContainer(s.service),
			http.WithHealthCheck[*http.ServiceOptions](opt.healthCheckFN),
		)
		if err != nil {
			return nil, err
		}
		s.helperService = h
	}
	return s, nil
}

func (s Service) Serve() error {
	logger := zerolog.Ctx(s.service.Ctx)
	if s.helperService != nil {
		logger.Info().Msgf("Starting helper service %d ", os.Getpid())
		go func() {
			if err := s.helperService.Serve(); err != nil {
				logger.Err(err).Msg("error helper service")
			}
		}()
	}
	logger.Info().Msgf("Starting process %d ", os.Getpid())
	s.service.Started()
	// [process] blocks until the context is closed
	err := s.process(s.service.Ctx)
	if err != nil {
		return err
	}
	return nil
}

// Setters
func WithName(n string) func(*Option) {
	return func(h *Option) {
		h.name = n
	}
}

func WithContext(ctx context.Context) func(*Option) {
	return func(h *Option) {
		h.ctx = ctx
	}
}

func WithServeHandler(s func(ctx context.Context) error) func(*Option) {
	return func(h *Option) {
		h.serveHandler = s
	}
}

func WithAddr(addr string) func(*Option) {
	return func(h *Option) {
		h.addr = addr
	}
}
