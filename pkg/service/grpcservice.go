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

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GrpcService struct {
	*Service
	grpcServer *grpc.Server
}

func NewGrpService(ctx context.Context, name string, desc *grpc.ServiceDesc, impl any) (*GrpcService, error) {
	svc := GrpcService{}
	svc.Service = NewService(ctx, name)
	s := grpc.NewServer()
	s.RegisterService(desc, impl)
	reflection.Register(s)
	svc.grpcServer = s
	return &svc, nil
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
