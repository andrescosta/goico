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
	service         *service.Service
	server          *http.Server
	healthCheckFunc HealthChkFn
	pool            *sync.Pool
}

type healthStatus struct {
	status  string
	details map[string]string
}

type initRoutesFn = func(context.Context, *mux.Router) error
type HealthChkFn = func(context.Context) (map[string]string, error)

type ServiceOptions struct {
	extras  *extrasOptions
	service *service.Service
}

type RouterOptions struct {
	addr         *string
	extras       *extrasOptions
	ctx          context.Context
	name         string
	initRoutesFn initRoutesFn
}

type extrasOptions struct {
	healthChkFn HealthChkFn
}

func NewWithWouter(opts ...func(*RouterOptions)) (*Service, error) {
	svc := &Service{}
	svc.initWithDefaults()

	opt := &RouterOptions{
		extras:       &extrasOptions{},
		ctx:          context.Background(),
		initRoutesFn: func(ctx context.Context, r *mux.Router) error { return nil },
		addr:         nil,
		name:         "",
	}

	for _, o := range opts {
		o(opt)
	}

	//
	s, err := service.New(
		service.WithName(opt.name),
		service.WithAddr(opt.addr),
		service.WithContext(opt.ctx),
		service.WithKind("rest"),
	)
	if err != nil {
		return nil, err
	}
	svc.service = s

	// Mux Router config
	router := svc.getRouter()

	//// routes initialization
	if err := opt.initRoutesFn(svc.service.Ctx, router); err != nil {
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
	svc.initWithDefaults()

	opt := &ServiceOptions{
		extras:  &extrasOptions{},
		service: nil,
	}
	for _, o := range opts {
		o(opt)
	}
	svc.service = opt.service
	// Mux Router config
	router := svc.getRouter()
	// add health check handler if provided
	svc.initHealthCheckFn(opt.extras.healthChkFn, router)
	return svc, nil
}

func (s *Service) Serve() error {
	logger := zerolog.Ctx(s.service.Ctx)
	logger.Info().Msgf("Starting process %d ", os.Getpid())
	if s.service.Addr == nil {
		return errors.New("the address was not configured")
	}

	listener, err := net.Listen("tcp", *s.service.Addr)
	s.server.BaseContext = func(l net.Listener) context.Context { return s.service.Ctx }
	if err != nil {
		return fmt.Errorf("net.Listen: failed to create listener on %s: %w", *s.service.Addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		<-s.service.Ctx.Done()
		logger.Debug().Msg("HTTP service: context closed")
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		logger.Debug().Msg("HTTP service: shutting down")
		errCh <- s.server.Shutdown(shutdownCtx)
	}()

	logger.Debug().Msgf("HTTP service: started on %s", *s.service.Addr)
	s.service.Started()

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
	WriteJSONBody(b, s.service.Metadata(), http.StatusOK, `{error:"error getting metadata"}`, w)
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

func WithContainer(svc *service.Service) func(*ServiceOptions) {
	return func(opts *ServiceOptions) {
		opts.service = svc
	}
}

func (s *Service) initHealthCheckFn(h HealthChkFn, r *mux.Router) {
	s.healthCheckFunc = h
	r.HandleFunc("/health", s.healthCheckHandler)
}

func (s *Service) setServer(r *mux.Router) {
	s.server = &http.Server{
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}
}

func (s *Service) getRouter() (r *mux.Router) {
	// Mux Router config
	r = mux.NewRouter()
	//// setting middlewares
	rf := RecoveryFunc{StackLevel: StackLevelFullStack}
	r.Use(rf.TryToRecover())
	r.Use(obs.GetLoggingMiddleware)
	s.service.OtelProvider.InstrRouter(s.service.Name, r)
	if env.AsBool("metadata.enabled", false) {
		r.HandleFunc("/meta", s.metadataHandler)
	}
	s.setServer(r)
	return
}

func (s *Service) initWithDefaults() {
	s.pool = &sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 1024))
		},
	}
}
