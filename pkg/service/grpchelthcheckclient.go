package service

import (
	"context"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type GrpcHelthCheckClient struct {
	serverAddr string
	conn       *grpc.ClientConn
	client     healthpb.HealthClient
}

func NewGrpcServerHelthCheckClient(host string) (*GrpcHelthCheckClient, error) {
	conn, err := Dial(host)
	if err != nil {
		return nil, err
	}
	client := healthpb.NewHealthClient(conn)
	return &GrpcHelthCheckClient{
		serverAddr: host,
		conn:       conn,
		client:     client,
	}, nil
}

func (c *GrpcHelthCheckClient) Close() {
	c.conn.Close()
}

func (c *GrpcHelthCheckClient) Check(ctx context.Context, name string) (healthpb.HealthCheckResponse_ServingStatus, error) {
	r, err := c.client.Check(ctx, &healthpb.HealthCheckRequest{Service: name})
	if err != nil {
		return 0, err
	}
	return r.Status, nil

}
