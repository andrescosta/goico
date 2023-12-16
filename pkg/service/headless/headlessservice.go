package headless

import (
	"context"
	"os"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/http"
	"github.com/rs/zerolog"
)

type Service struct {
	service     *service.Service
	serve       func(ctx context.Context) error
	metaService *http.Service
}

type Option struct {
	serveHandler func(ctx context.Context) error
	ctx          context.Context
	name         string
	addr         string
}

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
	s.serve = opt.serveHandler
	addr := opt.addr
	if addr == "" {
		addr = env.Env(opt.name+".addr", "")
	}
	if addr != "" {
		h, err := http.NewWithService(http.WithService(s.service))
		if err != nil {
			return nil, err
		}
		s.metaService = h
	}
	return s, nil
}

func (s Service) Start() error {
	logger := zerolog.Ctx(s.service.Ctx)
	if s.metaService != nil {
		logger.Info().Msgf("Starting obs process %d ", os.Getpid())
		go func() {
			if err := s.metaService.Start(); err != nil {
				logger.Err(err).Msg("error obs service")
			}
		}()
	}
	logger.Info().Msgf("Starting process %d ", os.Getpid())
	s.service.Started()
	err := s.serve(s.service.Ctx)
	if err != nil {
		return err
	}
	return nil
}
