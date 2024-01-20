package process

import (
	"context"
	"errors"
	"net"
	"os"

	"github.com/andrescosta/goico/pkg/option"
	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/http"
	"github.com/rs/zerolog"
)

type TypeFn func(ctx context.Context) error

type Service struct {
	base           *service.Base
	process        TypeFn
	sidecarService *http.Service
}

type Option interface {
	Apply(*Options)
}
type Options struct {
	serveHandler  TypeFn
	healthCheckFN http.HealthCheckFn
	ctx           context.Context
	name          string
	addr          *string
	enableSidecar bool
	listener      service.HTTPListener
}

func noop(_ context.Context) error { return nil }

func New(opts ...Option) (*Service, error) {
	opt := &Options{
		enableSidecar: true,
		ctx:           context.Background(),
		name:          "noop",
		addr:          nil,
		serveHandler:  noop,
	}
	for _, o := range opts {
		o.Apply(opt)
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
	s.base = service
	s.process = opt.serveHandler
	if opt.enableSidecar {
		// creates an HTTP service to serve metadata and health information over http
		h, err := http.NewSidecar(
			http.WithPrimaryService(s.base),
			http.WithHealthCheck[*http.SidecarOptions](opt.healthCheckFN),
			http.WithListener[*http.SidecarOptions](opt.listener),
		)
		if err != nil {
			return nil, err
		}
		s.sidecarService = h
	}
	return s, nil
}

func (s Service) Serve() error {
	l, err := service.DefaultHTTPListener.Listen(*s.base.Addr)
	if err != nil {
		return err
	}
	return s.DoServe(l)
}

func (s Service) DoServe(listener net.Listener) error {
	logger := zerolog.Ctx(s.base.Ctx)
	if s.sidecarService == nil {
		return errors.New("sidecar not enabled")
	}
	logger.Info().Msgf("Starting helper service %d ", os.Getpid())

	if s.sidecarService != nil {
		logger := zerolog.Ctx(s.base.Ctx)
		logger.Info().Msgf("Starting helper service %d ", os.Getpid())
		go func() {
			if err := s.sidecarService.DoServe(listener); err != nil {
				logger.Err(err).Msg("error helper service")
			}
		}()
	}

	logger.Info().Msgf("Starting process %d ", os.Getpid())
	s.base.Started()
	// [process] blocks until the context is closed
	err := s.process(s.base.Ctx)
	if err != nil {
		return err
	}
	return nil
}

// Setters
func WithName(n string) Option {
	return option.NewFuncOption(func(h *Options) {
		h.name = n
	})
}

func WithContext(ctx context.Context) Option {
	return option.NewFuncOption(func(h *Options) {
		h.ctx = ctx
	})
}

func WithEnableSidecar(enableSidecar bool) Option {
	return option.NewFuncOption(func(h *Options) {
		h.enableSidecar = enableSidecar
	})
}

func WithServeHandler(s func(ctx context.Context) error) Option {
	return option.NewFuncOption(func(h *Options) {
		h.serveHandler = s
	})
}

func WithHealthCheckFN(s http.HealthCheckFn) Option {
	return option.NewFuncOption(func(h *Options) {
		h.healthCheckFN = s
	})
}

func WithAddr(addr *string) Option {
	return option.NewFuncOption(func(h *Options) {
		h.addr = addr
	})
}

func WithSidecarListener(l service.HTTPListener) Option {
	return option.NewFuncOption(func(h *Options) {
		h.listener = l
	})
}
