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
	base            *service.ServiceBase
	server          *http.Server
	healthCheckFunc HealthChkFn
	doListenerFn    DoListenerFn
	pool            *sync.Pool
	URL             string
}

type healthStatus struct {
	status  string
	details map[string]string
}

type (
	initRoutesFn        = func(context.Context, *mux.Router) error
	HealthChkFn         = func(context.Context) (map[string]string, error)
	HttpServerBuilderFn = func(http.Handler) *http.Server
	DoListenerFn        = func(string) (net.Listener, error)
)

type ServiceOptions struct {
	extras *extrasOptions
	base   *service.ServiceBase
}

type RouterOptions struct {
	addr         *string
	extras       *extrasOptions
	ctx          context.Context
	name         string
	initRoutesFn initRoutesFn
}

type extrasOptions struct {
	healthChkFn         HealthChkFn
	httpServerBuilderFn HttpServerBuilderFn
	doListener          DoListenerFn
}

func New(opts ...func(*RouterOptions)) (*Service, error) {
	svc := &Service{}
	setDefaults(svc)

	opt := &RouterOptions{
		extras: &extrasOptions{
			httpServerBuilderFn: httpServerBuilderDefault,
			doListener:          doListenerDefault,
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
	b, err := service.New(
		service.WithName(opt.name),
		service.WithAddr(opt.addr),
		service.WithContext(opt.ctx),
		service.WithKind("rest"),
	)
	if err != nil {
		return nil, err
	}
	svc.base = b

	svc.doListenerFn = opt.extras.doListener

	// Mux Router config
	router := svc.buildRouter(opt.extras.httpServerBuilderFn)

	//// routes initialization
	if err := opt.initRoutesFn(svc.base.Ctx, router); err != nil {
		return nil, err
	}
	// add health check handler if provided
	if opt.extras.healthChkFn != nil {
		svc.initHealthCheckFn(opt.extras.healthChkFn, router)
	}
	return svc, nil
}

func NewWithServiceContainer(opts ...func(*ServiceOptions)) (*Service, error) {
	svc := &Service{}
	setDefaults(svc)

	opt := &ServiceOptions{
		extras: &extrasOptions{},
		base:   nil,
	}
	for _, o := range opts {
		o(opt)
	}
	svc.base = opt.base
	// Mux Router config
	router := svc.buildRouter(opt.extras.httpServerBuilderFn)
	// add health check handler if provided
	if opt.extras.healthChkFn != nil {
		svc.initHealthCheckFn(opt.extras.healthChkFn, router)
	}
	return svc, nil
}

func (s *Service) Serve() error {
	logger := zerolog.Ctx(s.base.Ctx)
	logger.Info().Msgf("Starting process %d ", os.Getpid())
	if s.base.Addr == nil {
		return errors.New("the address was not configured")
	}

	listener, err := s.doListenerFn(*s.base.Addr)
	s.server.BaseContext = func(l net.Listener) context.Context { return s.base.Ctx }
	if err != nil {
		return fmt.Errorf("net.Listen: failed to create listener on %s: %w", *s.base.Addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		<-s.base.Ctx.Done()
		logger.Debug().Msg("HTTP service: context closed")
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		logger.Debug().Msg("HTTP service: shutting down")
		errCh <- s.server.Shutdown(shutdownCtx)
	}()

	logger.Debug().Msgf("HTTP service: started on %s", *s.base.Addr)
	s.base.Started()

	s.URL = "http://" + listener.Addr().String()
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
	status.details = m
	var statusCode int
	if err != nil {
		status.status = "error"
		statusCode = http.StatusInternalServerError
	} else {
		status.status = "alive"
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

func (s *Service) initHealthCheckFn(h HealthChkFn, r *mux.Router) {
	s.healthCheckFunc = h
	r.HandleFunc("/health", s.healthCheckHandler)
}

func (s *Service) buildRouter(b HttpServerBuilderFn) (r *mux.Router) {
	// Mux Router config
	r = mux.NewRouter()
	//// setting middlewares
	rf := RecoveryFunc{StackLevel: StackLevelFullStack}
	r.Use(rf.TryToRecover())
	r.Use(obs.GetLoggingMiddleware)
	s.base.OtelProvider.InstrRouter(s.base.Name, r)
	if env.Bool("metadata.enabled", false) {
		r.HandleFunc("/meta", s.metadataHandler)
	}
	s.server = b(r)
	return
}
func httpServerBuilderDefault(r http.Handler) *http.Server {
	return &http.Server{
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      http.TimeoutHandler(r, time.Second, ""),
	}
}

func doListenerDefault(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}
func setDefaults(s *Service) {
	s.pool = &sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 1024))
		},
	}
}

// Setters
func WithInitRoutesFn(i initRoutesFn) func(*RouterOptions) {
	return func(w *RouterOptions) {
		w.initRoutesFn = i
	}
}

func WithAddr(a *string) func(*RouterOptions) {
	return func(r *RouterOptions) {
		r.addr = a
	}
}

func WithName(n string) func(*RouterOptions) {
	return func(r *RouterOptions) {
		r.name = n
	}
}

func WithContext(ctx context.Context) func(*RouterOptions) {
	return func(r *RouterOptions) {
		r.ctx = ctx
	}
}

func WithHealthCheck[T *ServiceOptions | *RouterOptions](healthChkFn HealthChkFn) func(T) {
	return func(t T) {
		switch v := any(t).(type) {
		case *ServiceOptions:
			v.extras.healthChkFn = healthChkFn
		case *RouterOptions:
			v.extras.healthChkFn = healthChkFn
		}
	}
}

func WithHttpServerBuilder[T *ServiceOptions | *RouterOptions](builder HttpServerBuilderFn) func(T) {
	return func(t T) {
		switch v := any(t).(type) {
		case *ServiceOptions:
			v.extras.httpServerBuilderFn = builder
		case *RouterOptions:
			v.extras.httpServerBuilderFn = builder
		}
	}
}

func WithDoListener[T *ServiceOptions | *RouterOptions](d DoListenerFn) func(T) {
	return func(t T) {
		switch v := any(t).(type) {
		case *ServiceOptions:
			v.extras.doListener = d
		case *RouterOptions:
			v.extras.doListener = d
		}
	}
}

func WithContainer(svc *service.ServiceBase) func(*ServiceOptions) {
	return func(opts *ServiceOptions) {
		opts.base = svc
	}
}
