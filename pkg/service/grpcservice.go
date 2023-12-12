package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/andrescosta/goico/pkg/service/svcmeta"
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

func NewGrpcService(ctx context.Context, name string, desc *grpc.ServiceDesc,
	initHandler func(context.Context) (any, error)) (*GrpcService, error) {
	return newGrpcService(ctx, name, desc, initHandler, nil)
}

func NewGrpcServiceWithHealthCheck(ctx context.Context, name string, desc *grpc.ServiceDesc,
	initHandler func(context.Context) (any, error),
	healthCheckHandler func(context.Context) error) (*GrpcService, error) {
	return newGrpcService(ctx, name, desc, initHandler, healthCheckHandler)
}

func newGrpcService(ctx context.Context, name string, desc *grpc.ServiceDesc,
	initHandler func(context.Context) (any, error),
	healthCheckHandler func(context.Context) error) (*GrpcService, error) {
	svc := GrpcService{}
	t := "grpc"
	s, err := newService(ctx, name, t)
	if err != nil {
		return nil, err
	}
	svc.Service = s
	var sopts []grpc.ServerOption
	sopts = append(sopts, s.otelProvider.InstrumentGrpcServer())
	server := grpc.NewServer(sopts...)

	h, err := initHandler(svc.ctx)
	if err != nil {
		return nil, err
	}

	ha, ok := h.(Closeable)
	if ok {
		svc.closeableHandler = ha
	}

	server.RegisterService(desc, h)
	reflection.Register(server)

	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(server, healthcheck)

	svcmeta.RegisterGrpcMetadataServer(server, NewGrpcServerInfo(&svc))

	svc.grpcServer = server

	healthcheck.SetServingStatus(name, healthpb.HealthCheckResponse_SERVING)
	if healthCheckHandler != nil {
		go check(ctx, name, healthcheck, healthCheckHandler)
	}
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

func (g *GrpcService) Info() map[string]string {
	return map[string]string{"Name": g.Name,
		"Addr":       *g.addr,
		"Start Time": g.startTime.String(),
		"Type":       "GRPC"}
}

func (g *GrpcService) Serve() error {
	logger := zerolog.Ctx(g.ctx)

	defer func() {
		if r := recover(); r != nil {
			logger.Fatal().Msg("error recovering")
		}
	}()

	logger.Info().Msgf("Starting process %d ", os.Getpid())

	if g.addr == nil {
		return ErrNotAddress
	}
	listener, err := net.Listen("tcp", *g.addr)
	if err != nil {
		return fmt.Errorf("failed to create listener on %s: %w", *g.addr, err)
	}

	go func() {
		<-g.ctx.Done()
		logger.Debug().Msg("GRPC Server: shutting down")
		g.grpcServer.GracefulStop()
	}()

	logger.Debug().Msgf("GRPC Server: started on %s", *g.addr)
	g.startTime = time.Now()
	if err := g.grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("failed to serve: %w", err)
	}

	logger.Debug().Msg("GRPC Server: stopped")
	logger.Info().Msgf("Process %d ended ", os.Getpid())
	return nil
}

func (g *GrpcService) Dispose() {
	if g.closeableHandler != nil {
		zerolog.Ctx(g.ctx).Debug().Msg("handler closed")
		g.closeableHandler.Close()
	}
}
