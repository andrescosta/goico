package process

import (
	"context"
	"net"
	"os"
	"sync"
	"time"

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
	svc := &Service{}
	base, err := service.New(
		service.WithName(opt.name),
		service.WithContext(opt.ctx),
		service.WithKind("headless"),
		service.WithAddr(opt.addr),
	)
	if err != nil {
		return nil, err
	}
	svc.Base = base
	svc.start = opt.starter
	// creates an HTTP service to serve metadata and health information over http
	sidecar, err := http.NewSidecar(
		http.WithPrimaryService(svc.Base),
		http.WithHealthCheckFn[*http.SidecarOptions](opt.healthCheckFN),
		http.WithListener[*http.SidecarOptions](opt.listener),
		http.WithServiceStatusFn(func() (string, time.Time) { return svc.Base.Status() }),
		http.WithInitRoutesFn[*http.SidecarOptions](func(_ context.Context, router *mux.Router) error {
			if opt.profilingEnabled {
				service.AttachProfilingHandlers(router)
			}
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	svc.sidecarService = sidecar
	return svc, nil
}

func (s Service) Serve() error {
	return s.ServeWithDelay(0)
}

func (s Service) ServeWithDelay(delay time.Duration) error {
	listener, err := s.sidecarService.Listener.Listen(s.Base.Addr)
	if err != nil {
		return err
	}
	return s.DoServeWithDelay(listener, delay)
}

func (s Service) DoServe(listener net.Listener) error {
	return s.DoServeWithDelay(listener, 0)
}

func (s Service) DoServeWithDelay(listener net.Listener, delay time.Duration) error {
	logger := zerolog.Ctx(s.Base.Ctx)
	logger.Info().Msgf("Starting helper service %d ", os.Getpid())
	var w sync.WaitGroup
	w.Add(2)
	logger.Info().Msgf("Starting process %d ", os.Getpid())
	var err error
	go func() {
		defer w.Done()
		sleep(s.Base.Ctx, delay)
		if s.Base.Ctx.Err() == nil {
			s.Base.SetStartedNow()
			err = s.start(s.Base.Ctx)
		}
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

func (s Service) NewHelthCheckClient(c service.HTTPClientBuilder) *http.HealthCheckClient {
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

func sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
		return
	case <-timer.C:
		return
	}
}
