package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/grpc/svcmeta"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type initHandler func(context.Context) (any, error)

type grpcOptions struct {
	addr               *string
	ctx                context.Context
	name               string
	initHandler        initHandler
	serviceDesc        *grpc.ServiceDesc
	healthCheckHandler func(context.Context) error
}

type Service struct {
	service          *service.Service
	grpcServer       *grpc.Server
	closeableHandler Closeable
}

type Closeable interface {
	Close() error
}

func New(opts ...func(*grpcOptions)) (*Service, error) {
	opt := &grpcOptions{
		ctx: context.Background(),
	}

	for _, o := range opts {
		o(opt)
	}

	g := &Service{}
	s, err := service.New(
		service.WithName(opt.name),
		service.WithAddr(opt.addr),
		service.WithContext(opt.ctx),
		service.WithKind("rest"),
	)
	if err != nil {
		return nil, err
	}
	g.service = s

	var sopts []grpc.ServerOption
	sopts = append(sopts, g.service.OtelProvider.InstrumentGrpcServer())
	server := grpc.NewServer(sopts...)
	h, err := opt.initHandler(g.service.Ctx)
	if err != nil {
		return nil, err
	}
	ha, ok := h.(Closeable)
	if ok {
		g.closeableHandler = ha
	}
	server.RegisterService(opt.serviceDesc, h)
	reflection.Register(server)
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(server, healthcheck)
	svcmeta.RegisterGrpcMetadataServer(server, svcmeta.NewServerInfo(g.Metadata()))
	g.grpcServer = server
	healthcheck.SetServingStatus(g.service.Name, healthpb.HealthCheckResponse_SERVING)
	if opt.healthCheckHandler != nil {
		go healthcheckIt(g.service.Ctx, g.service.Name, healthcheck, opt.healthCheckHandler)
	}
	return g, nil
}

func (g *Service) Serve() error {
	logger := zerolog.Ctx(g.service.Ctx)
	logger.Info().Msgf("Starting process %d ", os.Getpid())
	if g.service.Addr == nil {
		return errors.New("GrpcService.serve: the listening address was not configured")
	}

	listener, err := net.Listen("tcp", *g.service.Addr)
	if err != nil {
		return fmt.Errorf("net.listen: failed to create listener on %s: %w", *g.service.Addr, err)
	}
	go func() {
		<-g.service.Ctx.Done()
		logger.Debug().Msg("GRPC Server: shutting down")
		g.grpcServer.GracefulStop()
	}()
	logger.Debug().Msgf("GRPC Server: started on %s", *g.service.Addr)
	g.service.Started()
	if err := g.grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("failed to serve: %w", err)
	}
	logger.Debug().Msg("GRPC Server: stopped")
	logger.Info().Msgf("Process %d ended ", os.Getpid())
	return nil
}

func (g *Service) Dispose() {
	if g.closeableHandler != nil {
		zerolog.Ctx(g.service.Ctx).Debug().Msg("handler closed")
		g.closeableHandler.Close()
	}
}

func (g *Service) Metadata() map[string]string {
	return g.service.Metadata()
}

func healthcheckIt(ctx context.Context, name string, healthcheck *health.Server, healthCheckHandler func(context.Context) error) {
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

// Setters
func WithAddr(a *string) func(*grpcOptions) {
	return func(r *grpcOptions) {
		r.addr = a
	}
}

func WithHealthCheckHandler(h func(context.Context) error) func(*grpcOptions) {
	return func(r *grpcOptions) {
		r.healthCheckHandler = h
	}
}

func WithName(n string) func(*grpcOptions) {
	return func(r *grpcOptions) {
		r.name = n
	}
}

func WithContext(ctx context.Context) func(*grpcOptions) {
	return func(r *grpcOptions) {
		r.ctx = ctx
	}
}

func WithInitHandler(initHandler initHandler) func(*grpcOptions) {
	return func(r *grpcOptions) {
		r.initHandler = initHandler
	}
}

func WithServiceDesc(serviceDesc *grpc.ServiceDesc) func(*grpcOptions) {
	return func(r *grpcOptions) {
		r.serviceDesc = serviceDesc
	}
}
