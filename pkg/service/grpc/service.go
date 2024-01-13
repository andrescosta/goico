package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/service"
	"github.com/andrescosta/goico/pkg/service/grpc/svcmeta"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type (
	NewServiceFn  func(context.Context) (any, error)
	HealthCheckFn func(context.Context) error
)

type grpcOptions struct {
	addr        *string
	ctx         context.Context
	name        string
	newService  NewServiceFn
	serviceDesc *grpc.ServiceDesc
	healthCheck HealthCheckFn
}

type Service struct {
	base        *service.Base
	grpcServer  *grpc.Server
	closeableFn Closeable
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

	svc := &Service{}
	sb, err := service.New(
		service.WithName(opt.name),
		service.WithAddr(opt.addr),
		service.WithContext(opt.ctx),
		service.WithKind("grpc"),
	)
	if err != nil {
		return nil, err
	}
	svc.base = sb

	var sopts []grpc.ServerOption
	sopts = append(sopts, svc.base.OtelProvider.InstrumentGrpcServer())

	grpcPanicRecoveryHandler := func(p any) (err error) {
		return status.Errorf(codes.Internal, "%s", p)
	}
	sopts = append(sopts, grpc.ChainUnaryInterceptor(
		recovery.UnaryServerInterceptor(
			recovery.WithRecoveryHandler(grpcPanicRecoveryHandler))))

	server := grpc.NewServer(sopts...)
	grpcsvc, err := opt.newService(svc.base.Ctx)
	if err != nil {
		return nil, err
	}
	cl, ok := grpcsvc.(Closeable)
	if ok {
		svc.closeableFn = cl
	}
	server.RegisterService(opt.serviceDesc, grpcsvc)
	reflection.Register(server)
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(server, healthcheck)
	if env.Bool("metadata.enabled", false) {
		svcmeta.RegisterGrpcMetadataServer(server, svcmeta.NewServerInfo(svc.Metadata()))
	}
	svc.grpcServer = server
	healthcheck.SetServingStatus(svc.base.Name(), healthpb.HealthCheckResponse_SERVING)
	if opt.healthCheck != nil {
		go healthcheckIt(svc.base.Ctx, svc.base.Name(), healthcheck, opt.healthCheck)
	}
	return svc, nil
}

func (g *Service) Serve() error {
	listener, err := net.Listen("tcp", *g.base.Addr)
	if err != nil {
		return fmt.Errorf("net.listen: failed to create listener on %s: %w", *g.base.Addr, err)
	}
	return g.DoServe(listener)
}

func (g *Service) DoServe(listener net.Listener) error {
	logger := zerolog.Ctx(g.base.Ctx)
	logger.Info().Msgf("Starting process %d ", os.Getpid())
	if g.base.Addr == nil {
		return errors.New("GrpcService.serve: the listening address was not configured")
	}

	go func() {
		<-g.base.Ctx.Done()
		logger.Debug().Msg("GRPC Server: shutting down")
		g.grpcServer.GracefulStop()
	}()
	logger.Debug().Msgf("GRPC Server: started on %s", *g.base.Addr)
	g.base.Started()
	if err := g.grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("failed to serve: %w", err)
	}
	logger.Debug().Msg("GRPC Server: stopped")
	logger.Info().Msgf("Process %d ended ", os.Getpid())
	return nil
}

func (g *Service) Dispose() {
	logger := zerolog.Ctx(g.base.Ctx)
	if g.closeableFn != nil {
		logger.Debug().Msg("handler closed")
		if err := g.closeableFn.Close(); err != nil {
			logger.Err(err).Msg("grp service.Dispose: error closing resource")
		}
	}
}

func (g *Service) Metadata() map[string]string {
	return g.base.Metadata()
}

func healthcheckIt(ctx context.Context, name string, healthcheck *health.Server, healthCheckHandler func(context.Context) error) {
	current := healthpb.HealthCheckResponse_SERVING
	for {
		timer := time.NewTicker(*env.Duration("grpc.healthcheck", 5*time.Second))
		defer timer.Stop()
		select {
		case <-ctx.Done():
			healthcheck.Shutdown()
			return
		case <-timer.C:
			if err := healthCheckHandler(ctx); err != nil {
				if current == healthpb.HealthCheckResponse_SERVING {
					healthcheck.SetServingStatus(name, healthpb.HealthCheckResponse_NOT_SERVING)
					current = healthpb.HealthCheckResponse_NOT_SERVING
				}
			} else {
				if current == healthpb.HealthCheckResponse_NOT_SERVING {
					healthcheck.SetServingStatus(name, healthpb.HealthCheckResponse_SERVING)
					current = healthpb.HealthCheckResponse_SERVING
				}
			}
		}
	}
}

func (g *Service) Name() string {
	return g.base.Name()
}

func (g *Service) Addr() *string {
	return g.base.Addr
}

// Setters
func WithAddr(a *string) func(*grpcOptions) {
	return func(r *grpcOptions) {
		r.addr = a
	}
}

func WithHealthCheckFn(healthCheck HealthCheckFn) func(*grpcOptions) {
	return func(r *grpcOptions) {
		r.healthCheck = healthCheck
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

func WithNewServiceFn(newService NewServiceFn) func(*grpcOptions) {
	return func(r *grpcOptions) {
		r.newService = newService
	}
}

func WithServiceDesc(serviceDesc *grpc.ServiceDesc) func(*grpcOptions) {
	return func(r *grpcOptions) {
		r.serviceDesc = serviceDesc
	}
}
