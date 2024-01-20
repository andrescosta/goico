package grpc

import (
	"context"

	"github.com/andrescosta/goico/pkg/service"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type HelthCheckClient struct {
	serverAddr string
	conn       *grpc.ClientConn
	client     healthpb.HealthClient
}

func NewHelthCheckClient(ctx context.Context, addr string, d service.GrpcDialer) (*HelthCheckClient, error) {
	conn, err := d.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewHelthCheckClientWithConn(conn)
}

func NewHelthCheckClientWithConn(conn *grpc.ClientConn) (*HelthCheckClient, error) {
	conn.Target()
	client := healthpb.NewHealthClient(conn)
	return &HelthCheckClient{
		serverAddr: conn.Target(),
		conn:       conn,
		client:     client,
	}, nil
}

func (c *HelthCheckClient) Close() {
	_ = c.conn.Close()
}

func (c *HelthCheckClient) Check(ctx context.Context, name string) (healthpb.HealthCheckResponse_ServingStatus, error) {
	r, err := c.client.Check(ctx, &healthpb.HealthCheckRequest{Service: name})
	if err != nil {
		return healthpb.HealthCheckResponse_UNKNOWN, err
	}
	return r.Status, nil
}
