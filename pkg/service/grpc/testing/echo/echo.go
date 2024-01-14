package echo

import (
	context "context"
	"log"
	"net"

	"github.com/andrescosta/goico/pkg/service/grpc"
	"github.com/andrescosta/goico/pkg/service/grpc/svcmeta"
	rpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type Service struct {
	listener *bufconn.Listener
	service  *grpc.Service
	// broadcasterEcho *broadcaster.Broadcaster[*EchoResponse]
}

type Server struct {
	UnimplementedEchoServer
}

func New(ctx context.Context) (*Service, error) {
	return NewWithServer(ctx, EchoServer(&Server{}), nil)
}

func (s *Service) Name() string {
	return s.service.Name()
}

func (s *Service) Addr() *string {
	return s.service.Addr()
}

func NewWithServer(ctx context.Context, server EchoServer, h grpc.HealthCheckFn) (*Service, error) {
	buffer := 101024 * 1024
	l := bufconn.Listen(buffer)
	addr := "0.0.0.0"
	svc, err := grpc.New(
		grpc.WithName("echo"),
		grpc.WithAddr(&addr),
		grpc.WithHealthCheckFn(h),
		grpc.WithContext(ctx),
		grpc.WithServiceDesc(&Echo_ServiceDesc),
		grpc.WithNewServiceFn(func(ctx context.Context) (any, error) {
			return server, nil
		}),
	)
	if err != nil {
		log.Panicf("error starting ctl service: %s", err)
	}

	return &Service{
		listener: l,
		service:  svc,
	}, nil
}

func (s *Server) Echo(_ context.Context, req *EchoRequest) (*EchoResponse, error) {
	return &EchoResponse{
		Code:    req.Code,
		Message: req.Message,
	}, nil
}

func (s *Server) NoEcho(context.Context, *EchoRequest) (*Void, error) {
	return &Void{}, nil
}

func (s *Service) Serve() error {
	return s.service.DoServe(s.listener)
}

func (s *Service) Dispose() {
	s.service.Dispose()
}

func (s *Service) Client(ctx context.Context) (EchoClient, error) {
	conn, err := rpc.DialContext(ctx, "",
		rpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return s.listener.Dial()
		}), rpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		var t EchoClient
		return t, err
	}
	return NewEchoClient(conn), nil
}

func (s *Service) HealthCheckClient(ctx context.Context) (*grpc.HelthCheckClient, error) {
	conn, err := rpc.DialContext(ctx, "",
		rpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return s.listener.Dial()
		}), rpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return grpc.NewHelthCheckClientWithConn(conn)
}

func (s *Service) InfoClient(ctx context.Context) (*svcmeta.InfoClient, error) {
	conn, err := rpc.DialContext(ctx, "",
		rpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return s.listener.Dial()
		}), rpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return svcmeta.NewInfoClientWithConn(conn)
}
