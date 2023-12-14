package service

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
	"github.com/andrescosta/goico/pkg/service/httputils"
	"github.com/andrescosta/goico/pkg/service/obs"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type HttpService struct {
	*Service
	server          *http.Server
	healthCheckFunc healthCheckFunc
	pool            *sync.Pool
}

type healthStatus struct {
	status  string
	details map[string]string
}

type configureRoutesFunc = func(context.Context, *mux.Router) error
type healthCheckFunc = func(context.Context) (map[string]string, error)

func NewHttpServiceWithService(service *Service, healthCheckFunc healthCheckFunc) (*HttpService, error) {
	return newHttpServiceWithService(service, nil, healthCheckFunc)
}

func NewHttpService(ctx context.Context, name string, initHandler configureRoutesFunc) (*HttpService, error) {
	return newHttpService(ctx, name, initHandler, nil)
}

func NewHttpServiceWithHeathCheck(ctx context.Context, name string, initHandler configureRoutesFunc, healthCheckFunc healthCheckFunc) (*HttpService, error) {
	return newHttpService(ctx, name, initHandler, healthCheckFunc)
}

func newHttpService(ctx context.Context, name string, configureRoutes configureRoutesFunc, healthCheckFunc healthCheckFunc) (*HttpService, error) {
	s, err := newService(ctx, name, "rest")
	if err != nil {
		return nil, err
	}
	return newHttpServiceWithService(s, configureRoutes, healthCheckFunc)
}

func newHttpServiceWithService(service *Service, configureRoutes configureRoutesFunc, healthCheckFunc healthCheckFunc) (*HttpService, error) {
	svc := HttpService{
		pool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 1024))
			},
		},
		Service: service,
	}

	r := mux.NewRouter()

	// setting middlewares
	//// adds recovery middleware
	rf := RecoveryFunc{StackLevel: StackLevelFullStack}
	r.Use(rf.TryToRecover())
	//// adds logging middleware
	r.Use(obs.GetLoggingMiddleware)
	////Otel
	svc.otelProvider.InstrumentMuxRouter(svc.Name, r)

	if configureRoutes != nil {
		err := configureRoutes(svc.ctx, r)
		if err != nil {
			return nil, err
		}
	}

	svc.server = &http.Server{
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	// add health check handler if provided
	if healthCheckFunc != nil {
		svc.healthCheckFunc = healthCheckFunc
		r.HandleFunc("/health", svc.healthCheckHandler)
	}

	// add metadata handler if enabled
	if env.EnvAsBool("metadata.enabled", false) {
		r.HandleFunc("/meta", svc.metadataHandler)
	}

	return &svc, nil
}

func (s *HttpService) Serve() error {
	err := s.Start()
	return err
}

func (s *HttpService) Start() error {
	logger := zerolog.Ctx(s.ctx)
	logger.Info().Msgf("Starting process %d ", os.Getpid())
	if s.addr == nil {
		return ErrNotAddress
	}
	listener, err := net.Listen("tcp", *s.Service.addr)
	s.server.BaseContext = func(l net.Listener) context.Context { return s.ctx }
	if err != nil {
		return fmt.Errorf("failed to create listener on %s: %w", *s.Service.addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		<-s.ctx.Done()

		logger.Debug().Msg("HTTP server: context closed")
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()

		logger.Debug().Msg("HTTP server: shutting down")
		errCh <- s.server.Shutdown(shutdownCtx)
	}()

	logger.Debug().Msgf("HTTP server: started on %s", *s.Service.addr)
	s.Service.startTime = time.Now()
	if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to serve: %w", err)
	}

	logger.Debug().Msg("HTTP server: serving stopped")

	if err := <-errCh; err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	logger.Info().Msgf("Process %d ended ", os.Getpid())
	return nil
}

func (s *HttpService) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
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
	httputils.WriteJSONBody(b, status, statusCode, `{error:"error getting health check status"}`, w)
}

func (s *HttpService) metadataHandler(w http.ResponseWriter, r *http.Request) {
	addr := *s.addr
	if s.addr == nil {
		addr = "not configured"
	}
	m := map[string]string{"Name": s.Name,
		"Addr":       addr,
		"Start Time": s.startTime.Format(time.UnixDate),
		"Type":       s.Service.Type}
	b := s.pool.Get().(*bytes.Buffer)
	b.Reset()
	httputils.WriteJSONBody(b, m, http.StatusOK, `{error:"error getting metadata"}`, w)

}

type Recovery struct {
	logger *zerolog.Logger
}

func (r *Recovery) Println(i ...interface{}) {
	for _, ii := range i {
		r.logger.Error().Msgf("%v", ii)
	}
}
