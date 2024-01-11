package process

import (
	"context"
	"os"

	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/http"
	"github.com/rs/zerolog"
)

type TypeFn func(ctx context.Context) error

type Service struct {
	service        *service.Base
	process        TypeFn
	sidecarService *http.Service
}

type Option struct {
	serveHandler  TypeFn
	healthCheckFN http.HealthCheckFn
	ctx           context.Context
	name          string
	addr          *string
	enableSidecar bool
}

func noop(_ context.Context) error { return nil }

func New(opts ...func(*Option)) (*Service, error) {
	opt := &Option{
		enableSidecar: true,
		ctx:           context.Background(),
		name:          "noop",
		addr:          nil,
		serveHandler:  noop,
	}
	for _, o := range opts {
		o(opt)
	}
	s := &Service{}
	service, err := service.New(
		service.WithName(opt.name),
		service.WithContext(opt.ctx),
		service.WithKind("headless"),
		service.WithAddr(opt.addr),
	)
	if err != nil {
		return nil, err
	}
	s.service = service
	s.process = opt.serveHandler
	if opt.enableSidecar {
		// creates an HTTP service to serve metadata and health information over http
		h, err := http.NewSidecar(
			http.WithPrimaryService(s.service),
			http.WithHealthCheck[*http.SidecarOptions](opt.healthCheckFN),
		)
		if err != nil {
			return nil, err
		}
		s.sidecarService = h
	}
	return s, nil
}

func (s Service) Serve() error {
	logger := zerolog.Ctx(s.service.Ctx)
	if s.sidecarService != nil {
		logger.Info().Msgf("Starting helper service %d ", os.Getpid())
		go func() {
			if err := s.sidecarService.Serve(); err != nil {
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

func WithEnableSidecar(enableSidecar bool) func(*Option) {
	return func(h *Option) {
		h.enableSidecar = enableSidecar
	}
}

func WithServeHandler(s func(ctx context.Context) error) func(*Option) {
	return func(h *Option) {
		h.serveHandler = s
	}
}

func WithAddr(addr *string) func(*Option) {
	return func(h *Option) {
		h.addr = addr
	}
}
