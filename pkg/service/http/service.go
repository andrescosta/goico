package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/option"
	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/obs"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

const Kind = service.HTTP

type Service struct {
	base            *service.Base
	server          *http.Server
	healthCheckFunc HealthCheckFn
	pool            *sync.Pool
	imsidecar       bool
	Listener        service.HTTPListener
	serviceStatusFn ServiceStatus
	sidecarService  *Service
	pprofAddr       *string
}

type (
	initRoutesFn  = func(context.Context, *mux.Router) error
	HealthCheckFn = func(context.Context) (map[string]string, error)
	ServiceStatus = func() (string, time.Time)
)

type SidecarOptions struct {
	common          *commonOptions
	base            *service.Base
	disableOtel     bool
	serviceStatusFn ServiceStatus
}

type httpOptions interface {
	*ServiceOptions | *SidecarOptions
}

type Option[T httpOptions] interface {
	Apply(T)
}

type ServiceOptions struct {
	addr             string
	common           *commonOptions
	ctx              context.Context
	name             string
	profilingEnabled bool
	pprofAddr        *string
}

type commonOptions struct {
	healthChkFn       HealthCheckFn
	stackLevelOnError StackLevel
	listener          service.HTTPListener
	initRoutesFn      initRoutesFn
}

func New(opts ...Option[*ServiceOptions]) (*Service, error) {
	svc := &Service{}
	setDefaults(svc)

	opt := &ServiceOptions{
		common: &commonOptions{
			stackLevelOnError: StackLevelSimple,
			listener:          service.DefaultHTTPListener,
			initRoutesFn:      func(_ context.Context, _ *mux.Router) error { return nil },
		},
		ctx:  context.Background(),
		addr: "",
		name: "",
	}

	for _, o := range opts {
		o.Apply(opt)
	}

	//
	base, err := service.New(
		service.WithName(opt.name),
		service.WithAddr(opt.addr),
		service.WithContext(opt.ctx),
		service.WithKind("rest"),
	)
	if err != nil {
		return nil, err
	}
	svc.base = base
	svc.serviceStatusFn = func() (string, time.Time) { return svc.base.Status() }

	svc.Listener = opt.common.listener
	// Mux Router initialization
	router := svc.initializeRouter(opt)

	//// routes initialization
	if err := opt.common.initRoutesFn(svc.base.Ctx, router); err != nil {
		return nil, err
	}

	// Profiling sidecar
	if opt.profilingEnabled {
		if opt.pprofAddr == nil {
			return nil, errors.New("pprofAddr is nill and profilingEnabled is true")
		}

		baseSideCar, err := service.New(
			service.WithName("pprof"),
			service.WithAddr(*opt.pprofAddr),
			service.WithContext(opt.ctx),
			service.WithKind("pprof"),
		)
		if err != nil {
			return nil, err
		}
		sidecar, err := NewSidecar(
			WithPrimaryService(baseSideCar),
			WithListener[*SidecarOptions](opt.common.listener),
			WithInitRoutesFn[*SidecarOptions](func(_ context.Context, r *mux.Router) error {
				service.AttachProfilingHandlers(r)
				return nil
			}),
		)
		if err != nil {
			return nil, err
		}
		svc.sidecarService = sidecar
		svc.pprofAddr = opt.pprofAddr
	}
	return svc, nil
}

func NewSidecar(opts ...Option[*SidecarOptions]) (*Service, error) {
	svc := &Service{}
	setDefaults(svc)

	opt := &SidecarOptions{
		common: &commonOptions{
			stackLevelOnError: StackLevelSimple,
			listener:          service.DefaultHTTPListener,
			initRoutesFn:      func(_ context.Context, _ *mux.Router) error { return nil },
		},
		base: nil,
	}
	for _, o := range opts {
		o.Apply(opt)
	}
	svc.base = opt.base
	svc.imsidecar = true
	svc.Listener = opt.common.listener
	svc.serviceStatusFn = opt.serviceStatusFn

	// Mux Router initialization
	router := svc.initializeRouterSidecar(*opt)
	//// routes initialization
	if err := opt.common.initRoutesFn(svc.base.Ctx, router); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *Service) Serve() error {
	listener, err := s.Listener.Listen(s.base.Addr)
	if err != nil {
		return err
	}
	return s.DoServe(listener)
}

func (s *Service) DoServe(listener net.Listener) error {
	defer s.base.Stopped()
	logger := zerolog.Ctx(s.base.Ctx)
	logger.Info().Msgf("Starting process %d ", os.Getpid())

	s.server.BaseContext = func(_ net.Listener) context.Context { return s.base.Ctx }
	errCh := make(chan error, 1)
	go func() {
		<-s.base.Ctx.Done()
		logger.Debug().Msg("HTTP service: context closed")
		shutdownCtx, done := context.WithTimeout(context.Background(),
			*env.Duration("http.shutdown.timeout", time.Second*5))
		defer done()
		logger.Debug().Msg("HTTP service: shutting down")
		errCh <- s.server.Shutdown(shutdownCtx)
	}()

	var w sync.WaitGroup
	if s.sidecarService != nil {
		w.Add(1)
		go func() {
			defer w.Done()
			if err := s.sidecarService.Serve(); err != nil {
				logger.Err(err).Msg("error helper service")
			}
		}()
	}

	logger.Debug().Msgf("HTTP service: started on %s", listener.Addr().String())

	if !s.imsidecar {
		s.base.SetStartedNow()
	}

	if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http.Serve: failed to serve: %w", err)
	}
	logger.Debug().Msg("HTTP service: serving stopped")
	if err := <-errCh; err != nil {
		return fmt.Errorf("http.Shutdown: failed to shutdown server: %w", err)
	}
	w.Wait()
	logger.Info().Msgf("Process %d ended ", os.Getpid())
	return nil
}

func (s *Service) RunHealthCheck(ctx context.Context) (map[string]string, error) {
	return s.healthCheckFunc(ctx)
}

func (s *Service) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/JSON")
	status := &service.HealthStatus{}
	m, err := s.RunHealthCheck(r.Context())
	status.Details = m
	var httpStatusCode int
	currStatus, when := s.serviceStatusFn()
	status.ServiceStatus = currStatus
	status.StartedAt = when.Format(time.UnixDate)
	if err != nil {
		status.Status = "ERROR"
		httpStatusCode = http.StatusInternalServerError
	} else {
		status.Status = "OK"
		httpStatusCode = http.StatusOK
	}
	buff := s.pool.Get().(*bytes.Buffer)
	buff.Reset()
	WriteJSONBody(buff, status, httpStatusCode, `{error:"error getting health check status"}`, w)
}

func (s *Service) metadataHandler(w http.ResponseWriter, _ *http.Request) {
	b := s.pool.Get().(*bytes.Buffer)
	b.Reset()
	WriteJSONBody(b, s.base.Metadata(), http.StatusOK, `{error:"error getting metadata"}`, w)
}

func (s *Service) initializeRouter(opts *ServiceOptions) (router *mux.Router) {
	// Mux Router config
	router = mux.NewRouter()
	//// setting middlewares
	rf := RecoveryFunc{StackLevel: opts.common.stackLevelOnError}
	router.Use(rf.TryToRecover())
	router.Use(obs.GetLoggingMiddleware)
	s.base.OtelProvider.InstrRouter(s.base.Name(), router)
	if env.Bool("metadata.enabled", false) {
		router.HandleFunc("/meta", s.metadataHandler)
	}
	if opts.common.healthChkFn != nil {
		s.healthCheckFunc = opts.common.healthChkFn
		router.HandleFunc("/health", s.healthCheckHandler)
	}
	s.server = newHTTPServer(router)
	return
}

func (s *Service) initializeRouterSidecar(opt SidecarOptions) (router *mux.Router) {
	// Mux Router config
	router = mux.NewRouter()
	//// setting middlewares
	rf := RecoveryFunc{StackLevel: opt.common.stackLevelOnError}
	router.Use(rf.TryToRecover())
	if env.Bool("metadata.enabled.sidecar", false) {
		router.HandleFunc("/meta", s.metadataHandler)
	}
	if opt.common.healthChkFn != nil {
		s.healthCheckFunc = opt.common.healthChkFn
		router.HandleFunc("/health", s.healthCheckHandler)
	}
	s.server = newHTTPServer(router)
	return
}

func (s *Service) HelthCheckClient(c service.HTTPClientBuilder) *HealthCheckClient {
	return &HealthCheckClient{
		ServerAddr: s.base.Addr,
		Builder:    c,
	}
}

func newHTTPServer(r http.Handler) *http.Server {
	return &http.Server{
		WriteTimeout: *env.Duration("http.timeout.write", time.Second*20),
		ReadTimeout:  *env.Duration("http.timeout.read", time.Second*20),
		IdleTimeout:  *env.Duration("http.timeout.idle", time.Second*20),
		Handler:      http.TimeoutHandler(r, *env.Duration("http.timeout.handler", 20*time.Second), ""),
	}
}

func setDefaults(s *Service) {
	s.pool = &sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 1024))
		},
	}
}

// Setters
func WithInitRoutesFn[T *SidecarOptions | *ServiceOptions](i initRoutesFn) Option[T] {
	return option.NewFuncOption(func(t T) {
		if t != nil {
			switch v := any(t).(type) {
			case *SidecarOptions:
				v.common.initRoutesFn = i
			case *ServiceOptions:
				v.common.initRoutesFn = i
			}
		}
	})
}

func WithAddr(a string) Option[*ServiceOptions] {
	return option.NewFuncOption(func(s *ServiceOptions) {
		s.addr = a
	})
}

func WithName(n string) Option[*ServiceOptions] {
	return option.NewFuncOption(func(s *ServiceOptions) {
		s.name = n
	})
}

func WithContext(ctx context.Context) Option[*ServiceOptions] {
	return option.NewFuncOption(func(s *ServiceOptions) {
		s.ctx = ctx
	})
}

func WithStackLevelOnError[T *SidecarOptions | *ServiceOptions](lvl StackLevel) Option[T] {
	return option.NewFuncOption(func(t T) {
		if t != nil {
			switch v := any(t).(type) {
			case *SidecarOptions:
				v.common.stackLevelOnError = lvl
			case *ServiceOptions:
				v.common.stackLevelOnError = lvl
			}
		}
	})
}

func WithHealthCheckFn[T *SidecarOptions | *ServiceOptions](healthChkFn HealthCheckFn) Option[T] {
	return option.NewFuncOption(func(t T) {
		if t != nil {
			switch v := any(t).(type) {
			case *SidecarOptions:
				v.common.healthChkFn = healthChkFn
			case *ServiceOptions:
				v.common.healthChkFn = healthChkFn
			}
		}
	})
}

func WithListener[T *SidecarOptions | *ServiceOptions](listener service.HTTPListener) Option[T] {
	return option.NewFuncOption(func(t T) {
		if t != nil && listener != nil {
			switch v := any(t).(type) {
			case *SidecarOptions:
				v.common.listener = listener
			case *ServiceOptions:
				v.common.listener = listener
			}
		}
	})
}

func WithPrimaryService(svc *service.Base) Option[*SidecarOptions] {
	return option.NewFuncOption(func(opts *SidecarOptions) {
		opts.base = svc
	})
}

func WithDisableOtel(otel bool) Option[*SidecarOptions] {
	return option.NewFuncOption(func(opts *SidecarOptions) {
		opts.disableOtel = otel
	})
}

func WithServiceStatusFn(f ServiceStatus) Option[*SidecarOptions] {
	return option.NewFuncOption(func(opts *SidecarOptions) {
		opts.serviceStatusFn = f
	})
}

func WithProfilingEnabled(p bool) Option[*ServiceOptions] {
	return option.NewFuncOption(func(s *ServiceOptions) {
		s.profilingEnabled = p
	})
}

func WithPProfAddr(addr *string) Option[*ServiceOptions] {
	return option.NewFuncOption(func(r *ServiceOptions) {
		r.pprofAddr = addr
	})
}
