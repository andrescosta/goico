package process

import (
	"context"
	"net"
	"net/http/pprof"
	"os"
	"sync"

	"github.com/andrescosta/goico/pkg/option"
	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/http"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

const Kind = service.Process

type StartFn func(ctx context.Context) error

type Service struct {
	Base           *service.Base
	start          StartFn
	sidecarService *http.Service
}

type Option interface {
	Apply(*Options)
}
type Options struct {
	starter          StartFn
	healthCheckFN    http.HealthCheckFn
	ctx              context.Context
	name             string
	addr             string
	listener         service.HTTPListener
	profilingEnabled bool
}

func noop(_ context.Context) error { return nil }

func New(opts ...Option) (*Service, error) {
	opt := &Options{
		ctx:     context.Background(),
		name:    "noop",
		addr:    "",
		starter: noop,
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
	s.Base = service
	s.start = opt.starter
	// creates an HTTP service to serve metadata and health information over http
	sidecar, err := http.NewSidecar(
		http.WithPrimaryService(s.Base),
		http.WithHealthCheck[*http.SidecarOptions](opt.healthCheckFN),
		http.WithListener[*http.SidecarOptions](opt.listener),
		http.WithInitRoutesFn[*http.SidecarOptions](func(_ context.Context, router *mux.Router) error {
			if opt.profilingEnabled {
				router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
			}
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	s.sidecarService = sidecar
	return s, nil
}

func (s Service) Serve() error {
	listener, err := s.sidecarService.Listener.Listen(s.Base.Addr)
	if err != nil {
		return err
	}
	return s.DoServe(listener)
}

func (s Service) DoServe(listener net.Listener) error {
	logger := zerolog.Ctx(s.Base.Ctx)
	logger.Info().Msgf("Starting helper service %d ", os.Getpid())
	var w sync.WaitGroup
	w.Add(2)
	logger.Info().Msgf("Starting process %d ", os.Getpid())
	s.Base.SetStartedNow()
	var err error
	go func() {
		defer w.Done()
		err = s.start(s.Base.Ctx)
		s.Base.Stop()
	}()
	go func() {
		defer w.Done()
		if err := s.sidecarService.DoServe(listener); err != nil {
			logger.Err(err).Msg("error helper service")
		}
	}()
	w.Wait()
	return err
}

func (s Service) HelthCheckClient(c service.HTTPClient) *http.HealthCheckClient {
	return &http.HealthCheckClient{
		ServerAddr: s.Base.Addr,
		Builder:    c,
	}
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

func WithStarter(s func(ctx context.Context) error) Option {
	return option.NewFuncOption(func(h *Options) {
		h.starter = s
	})
}

func WithHealthCheckFN(s http.HealthCheckFn) Option {
	return option.NewFuncOption(func(h *Options) {
		h.healthCheckFN = s
	})
}

func WithAddr(addr string) Option {
	return option.NewFuncOption(func(h *Options) {
		h.addr = addr
	})
}

func WithSidecarListener(l service.HTTPListener) Option {
	return option.NewFuncOption(func(h *Options) {
		h.listener = l
	})
}

func WithProfilingEnabled(p bool) Option {
	return option.NewFuncOption(func(s *Options) {
		s.profilingEnabled = p
	})
}
