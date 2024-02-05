package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/option"
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

const Kind = service.GRPC

type Option interface {
	Apply(*Options)
}
type Options struct {
	addr        string
	ctx         context.Context
	name        string
	newService  NewServiceFn
	serviceDesc *grpc.ServiceDesc
	healthCheck HealthCheckFn
	listener    service.GrpcListener
}

type Service struct {
	base           *service.Base
	grpcServer     *grpc.Server
	listener       service.GrpcListener
	healthCheck    HealthCheckFn
	healthCheckSrv *health.Server
	closeableFn    Closeable
}

type Closeable interface {
	Close() error
}

func New(opts ...Option) (*Service, error) {
	opt := &Options{
		ctx: context.Background(),
	}

	for _, o := range opts {
		o.Apply(opt)
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

	svc.listener = opt.listener

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
	if env.Bool("metadata.enabled", false) {
		svcmeta.RegisterGrpcMetadataServer(server, svcmeta.NewServerInfo(svc.Metadata()))
	}
	svc.grpcServer = server
	if opt.healthCheck != nil {
		healthCheckSrv := health.NewServer()
		healthpb.RegisterHealthServer(server, healthCheckSrv)
		healthCheckSrv.SetServingStatus(svc.base.Name(), healthpb.HealthCheckResponse_NOT_SERVING)
		svc.healthCheck = opt.healthCheck
		svc.healthCheckSrv = healthCheckSrv
	}
	return svc, nil
}

func (g *Service) Serve() error {
	listener, err := g.listener.Listen(g.base.Addr)
	if err != nil {
		return err
	}
	return g.DoServe(listener)
}

func (g *Service) DoServe(listener net.Listener) error {
	logger := zerolog.Ctx(g.base.Ctx)
	logger.Info().Msgf("Starting process %d ", os.Getpid())
	if g.base.Addr == "" {
		return errors.New("GrpcService.serve: the listening address was not configured")
	}

	go func() {
		<-g.base.Ctx.Done()
		logger.Debug().Msg("GRPC Server: shutting down")
		if g.healthCheckSrv != nil {
			g.healthCheckSrv.Shutdown()
		}
		g.grpcServer.GracefulStop()
	}()
	logger.Debug().Msgf("GRPC Server: started on %s", g.base.Addr)
	g.base.Started()

	if g.healthCheck != nil {
		go g.healthcheckIt(g.base.Ctx)
	}

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

func (g *Service) healthcheckIt(ctx context.Context) {
	current := g.checkAndSetServingStatus(ctx, healthpb.HealthCheckResponse_UNKNOWN)
	for {
		timer := time.NewTicker(*env.Duration("grpc.healthcheck", 5*time.Second))
		defer timer.Stop()
		select {
		case <-ctx.Done():
			g.healthCheckSrv.Shutdown()
			return
		case <-timer.C:
			current = g.checkAndSetServingStatus(ctx, current)
		}
	}
}

func (g *Service) checkAndSetServingStatus(ctx context.Context, current healthpb.HealthCheckResponse_ServingStatus) healthpb.HealthCheckResponse_ServingStatus {
	if err := g.healthCheck(ctx); err != nil {
		if current == healthpb.HealthCheckResponse_SERVING || current == healthpb.HealthCheckResponse_UNKNOWN {
			g.healthCheckSrv.SetServingStatus(g.base.Name(), healthpb.HealthCheckResponse_NOT_SERVING)
			return healthpb.HealthCheckResponse_NOT_SERVING
		}
	} else {
		if current == healthpb.HealthCheckResponse_NOT_SERVING || current == healthpb.HealthCheckResponse_UNKNOWN {
			g.healthCheckSrv.SetServingStatus(g.base.Name(), healthpb.HealthCheckResponse_SERVING)
			return healthpb.HealthCheckResponse_SERVING
		}
	}
	return current
}

func (g *Service) Name() string {
	return g.base.Name()
}

func (g *Service) Addr() string {
	return g.base.Addr
}

func (g *Service) Dial(d service.GrpcDialer) (*grpc.ClientConn, error) {
	addr := g.Addr()
	return d.Dial(g.base.Ctx, addr)
}

func (g *Service) HelthCheckClient(d service.GrpcDialer) (*HealthCheckClient, error) {
	conn, err := g.Dial(d)
	if err != nil {
		return nil, err
	}
	return NewHelthCheckClientWithConn(conn, g.Name())
}

// Setters
func WithAddr(a string) Option {
	return option.NewFuncOption(func(r *Options) {
		r.addr = a
	})
}

func WithHealthCheckFn(healthCheck HealthCheckFn) Option {
	return option.NewFuncOption(func(r *Options) {
		r.healthCheck = healthCheck
	})
}

func WithName(n string) Option {
	return option.NewFuncOption(func(r *Options) {
		r.name = n
	})
}

func WithContext(ctx context.Context) Option {
	return option.NewFuncOption(func(r *Options) {
		r.ctx = ctx
	})
}

func WithNewServiceFn(newService NewServiceFn) Option {
	return option.NewFuncOption(func(r *Options) {
		r.newService = newService
	})
}

func WithServiceDesc(serviceDesc *grpc.ServiceDesc) Option {
	return option.NewFuncOption(func(r *Options) {
		r.serviceDesc = serviceDesc
	})
}

func WithListener(l service.GrpcListener) Option {
	return option.NewFuncOption(func(r *Options) {
		r.listener = l
	})
}
