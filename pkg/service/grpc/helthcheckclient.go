package grpc

import (
	"context"

	"github.com/andrescosta/goico/pkg/service"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type HealthCheckClient struct {
	serverAddr string
	conn       *grpc.ClientConn
	name       string
	client     healthpb.HealthClient
}

func NewHelthCheckClient(ctx context.Context, addr string, name string, d service.GrpcDialer) (*HealthCheckClient, error) {
	conn, err := d.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewHelthCheckClientWithConn(conn, name)
}

func NewHelthCheckClientWithConn(conn *grpc.ClientConn, name string) (*HealthCheckClient, error) {
	conn.Target()
	client := healthpb.NewHealthClient(conn)
	return &HealthCheckClient{
		serverAddr: conn.Target(),
		conn:       conn,
		client:     client,
		name:       name,
	}, nil
}

func (c *HealthCheckClient) Close() error {
	return c.conn.Close()
}

func (c *HealthCheckClient) Check(ctx context.Context) (healthpb.HealthCheckResponse_ServingStatus, error) {
	r, err := c.client.Check(ctx, &healthpb.HealthCheckRequest{Service: c.name})
	if err != nil {
		return healthpb.HealthCheckResponse_UNKNOWN, err
	}
	return r.Status, nil
}

func (c *HealthCheckClient) CheckOk(ctx context.Context) error {
	r, err := c.Check(ctx)
	if err != nil {
		return err
	}
	if r != healthpb.HealthCheckResponse_SERVING {
		return service.ErrNotHealthy{Addr: c.serverAddr}
	}
	return nil
}
