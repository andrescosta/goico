package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type HttpService struct {
	*Service
	server *http.Server
}

func NewHttpService(ctx context.Context, name, resource string, initHandler func(context.Context) chi.Router) *HttpService {
	svc := HttpService{}
	svc.Service = NewService(ctx, name)
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Mount("/"+resource, initHandler(ctx))
	svc.server = &http.Server{Handler: router}
	return &svc
}

func (sh *HttpService) Serve() error {
	logger := zerolog.Ctx(sh.ctx)
	ctx, done := signal.NotifyContext(sh.ctx, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		done()
		if r := recover(); r != nil {
			logger.Fatal().Msg("error recovering")
		}
	}()

	logger.Info().Msgf("Starting process %d ", os.Getpid())

	listener, err := net.Listen("tcp", sh.Service.Addr)
	if err != nil {
		return fmt.Errorf("failed to create listener on %s: %w", sh.Service.Addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()

		logger.Debug().Msg("HTTP server: context closed")
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()

		logger.Debug().Msg("HTTP server: shutting down")
		errCh <- sh.server.Shutdown(shutdownCtx)
	}()

	logger.Debug().Msgf("HTTP server: started on %s", sh.Service.Addr)
	sh.Service.StartTime = time.Now()
	if err := sh.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to serve: %w", err)
	}

	logger.Debug().Msg("HTTP server: serving stopped")

	if err := <-errCh; err != nil {
		done()
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	done()
	logger.Info().Msgf("Process %d ended ", os.Getpid())
	return nil
}
