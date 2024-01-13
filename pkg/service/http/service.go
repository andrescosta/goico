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
	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/obs"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type Service struct {
	base            *service.Base
	server          *http.Server
	healthCheckFunc HealthCheckFn
	pool            *sync.Pool
}

type healthStatus struct {
	Status  string            `json:"status"`
	Details map[string]string `json:"details"`
}

type (
	initRoutesFn  = func(context.Context, *mux.Router) error
	HealthCheckFn = func(context.Context) (map[string]string, error)
)

type SidecarOptions struct {
	common *commonOptions
	base   *service.Base
}

type ServiceOptions struct {
	addr         *string
	common       *commonOptions
	ctx          context.Context
	name         string
	initRoutesFn initRoutesFn
}

type commonOptions struct {
	healthChkFn       HealthCheckFn
	stackLevelOnError StackLevel
}

func New(opts ...func(*ServiceOptions)) (*Service, error) {
	svc := &Service{}
	setDefaults(svc)

	opt := &ServiceOptions{
		common: &commonOptions{
			stackLevelOnError: StackLevelSimple,
		},
		ctx:          context.Background(),
		initRoutesFn: func(ctx context.Context, r *mux.Router) error { return nil },
		addr:         nil,
		name:         "",
	}

	for _, o := range opts {
		o(opt)
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

	// Mux Router initialization
	router := svc.initializeRouter(opt.common)

	//// routes initialization
	if err := opt.initRoutesFn(svc.base.Ctx, router); err != nil {
		return nil, err
	}
	return svc, nil
}

func NewSidecar(opts ...func(*SidecarOptions)) (*Service, error) {
	svc := &Service{}
	setDefaults(svc)

	opt := &SidecarOptions{
		common: &commonOptions{},
		base:   nil,
	}
	for _, o := range opts {
		o(opt)
	}
	svc.base = opt.base

	// Mux Router initialization
	_ = svc.initializeRouter(opt.common)
	return svc, nil
}

func (s *Service) Serve() error {
	listener, err := net.Listen("tcp", *s.base.Addr)
	if err != nil {
		return fmt.Errorf("net.Listen: failed to create listener on %s: %w", *s.base.Addr, err)
	}
	return s.DoServe(listener)
}

func (s *Service) DoServe(listener net.Listener) error {
	defer s.base.Stopped()
	logger := zerolog.Ctx(s.base.Ctx)
	logger.Info().Msgf("Starting process %d ", os.Getpid())

	s.server.BaseContext = func(l net.Listener) context.Context { return s.base.Ctx }
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

	logger.Debug().Msgf("HTTP service: started on %s", *s.base.Addr)
	s.base.Started()

	if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http.Serve: failed to serve: %w", err)
	}
	logger.Debug().Msg("HTTP service: serving stopped")
	if err := <-errCh; err != nil {
		return fmt.Errorf("http.Shutdown: failed to shutdown server: %w", err)
	}
	logger.Info().Msgf("Process %d ended ", os.Getpid())
	return nil
}

func (s *Service) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/JSON")
	status := &healthStatus{}
	m, err := s.healthCheckFunc(r.Context())
	status.Details = m
	var statusCode int
	if err != nil {
		status.Status = "error"
		statusCode = http.StatusInternalServerError
	} else {
		status.Status = "alive"
		statusCode = http.StatusOK
	}
	b := s.pool.Get().(*bytes.Buffer)
	b.Reset()
	WriteJSONBody(b, status, statusCode, `{error:"error getting health check status"}`, w)
}

func (s *Service) metadataHandler(w http.ResponseWriter, _ *http.Request) {
	b := s.pool.Get().(*bytes.Buffer)
	b.Reset()
	WriteJSONBody(b, s.base.Metadata(), http.StatusOK, `{error:"error getting metadata"}`, w)
}

func (s *Service) initializeRouter(opts *commonOptions) (r *mux.Router) {
	// Mux Router config
	r = mux.NewRouter()
	//// setting middlewares
	rf := RecoveryFunc{StackLevel: opts.stackLevelOnError}
	r.Use(rf.TryToRecover())
	r.Use(obs.GetLoggingMiddleware)
	s.base.OtelProvider.InstrRouter(s.base.Name(), r)
	if env.Bool("metadata.enabled", false) {
		r.HandleFunc("/meta", s.metadataHandler)
	}
	if opts.healthChkFn != nil {
		s.healthCheckFunc = opts.healthChkFn
		r.HandleFunc("/health", s.healthCheckHandler)
	}
	s.server = newHTTPServer(r)
	return
}

func newHTTPServer(r http.Handler) *http.Server {
	return &http.Server{
		WriteTimeout: *env.Duration("http.timeout.write", time.Second*5),
		ReadTimeout:  *env.Duration("http.timeout.read", time.Second*5),
		IdleTimeout:  *env.Duration("http.timeout.idle", time.Second*5),
		Handler:      http.TimeoutHandler(r, *env.Duration("http.timeout.handler", time.Second), ""),
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
func WithInitRoutesFn(i initRoutesFn) func(*ServiceOptions) {
	return func(w *ServiceOptions) {
		w.initRoutesFn = i
	}
}

func WithAddr(a *string) func(*ServiceOptions) {
	return func(r *ServiceOptions) {
		r.addr = a
	}
}

func WithName(n string) func(*ServiceOptions) {
	return func(r *ServiceOptions) {
		r.name = n
	}
}

func WithContext(ctx context.Context) func(*ServiceOptions) {
	return func(r *ServiceOptions) {
		r.ctx = ctx
	}
}

func WithStackLevelOnError[T *SidecarOptions | *ServiceOptions](lvl StackLevel) func(T) {
	return func(t T) {
		if t != nil {
			switch v := any(t).(type) {
			case *SidecarOptions:
				v.common.stackLevelOnError = lvl
			case *ServiceOptions:
				v.common.stackLevelOnError = lvl
			}
		}
	}
}

func WithHealthCheck[T *SidecarOptions | *ServiceOptions](healthChkFn HealthCheckFn) func(T) {
	return func(t T) {
		if t != nil {
			switch v := any(t).(type) {
			case *SidecarOptions:
				v.common.healthChkFn = healthChkFn
			case *ServiceOptions:
				v.common.healthChkFn = healthChkFn
			}
		}
	}
}

func WithPrimaryService(svc *service.Base) func(*SidecarOptions) {
	return func(opts *SidecarOptions) {
		opts.base = svc
	}
}
