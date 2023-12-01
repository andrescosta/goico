package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	info "github.com/andrescosta/goico/pkg/service/info/grpc"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/reflection"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type GrpcService struct {
	*Service
	grpcServer       *grpc.Server
	closeableHandler Closeable
}

type Closeable interface {
	Close() error
}

func EmptyhealthCheckHandler(context.Context) error {
	return nil
}

func NewGrpService(ctx context.Context, name string, desc *grpc.ServiceDesc,
	initHandler func(context.Context) (any, error),
	healthCheckHandler func(context.Context) error) (*GrpcService, error) {
	svc := GrpcService{}
	svc.Service = NewService(ctx, name)
	var sopts []grpc.ServerOption

	// sopts = append(sopts, grpc.StatsHandler(&Handler{}))
	// sopts = append(sopts, grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	//"go.opencensus.io/plugin/ocgrpc"

	s := grpc.NewServer(sopts...)

	h, err := initHandler(svc.ctx)
	if err != nil {
		return nil, err
	}

	ha, ok := h.(Closeable)
	if ok {
		svc.closeableHandler = ha
	}

	s.RegisterService(desc, h)
	reflection.Register(s)

	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(s, healthcheck)

	info.RegisterSvcInfoServer(s, NewGrpcServerInfo(&svc))

	svc.grpcServer = s

	healthcheck.SetServingStatus(name, healthpb.HealthCheckResponse_SERVING)

	go check(ctx, name, healthcheck, healthCheckHandler)

	return &svc, nil
}

func check(ctx context.Context, name string, healthcheck *health.Server, healthCheckHandler func(context.Context) error) {
	next := healthpb.HealthCheckResponse_SERVING
	for {
		timer := time.NewTicker(5 * time.Second)
		select {
		case <-ctx.Done():
			healthcheck.Shutdown()
			return
		case <-timer.C:
			if err := healthCheckHandler(ctx); err != nil {
				healthcheck.SetServingStatus(name, healthpb.HealthCheckResponse_NOT_SERVING)
				next = healthpb.HealthCheckResponse_NOT_SERVING
			} else {
				if next == healthpb.HealthCheckResponse_NOT_SERVING {
					healthcheck.SetServingStatus(name, healthpb.HealthCheckResponse_SERVING)
				}
			}
		}
	}
}

func (sh *GrpcService) Info() map[string]string {
	return map[string]string{"Name": sh.Name,
		"Addr":       sh.Addr,
		"Start Time": sh.StartTime.String(),
		"Type":       "GRPC"}
}

func (sh *GrpcService) Serve() error {
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

	go func() {
		<-ctx.Done()
		logger.Debug().Msg("GRPC Server: shutting down")
		sh.grpcServer.GracefulStop()
	}()

	logger.Debug().Msgf("GRPC Server: started on %s", sh.Addr)
	sh.StartTime = time.Now()
	if err := sh.grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("failed to serve: %w", err)
	}
	logger.Debug().Msg("GRPC Server: stopped")

	done()
	logger.Info().Msgf("Process %d ended ", os.Getpid())
	return nil
}

func (sh *GrpcService) Dispose() {
	if sh.closeableHandler != nil {
		zerolog.Ctx(sh.ctx).Debug().Msg("handler closed")
		sh.closeableHandler.Close()
	}
}
