package service

import (
	"context"

	pb "github.com/andrescosta/goico/pkg/service/info/grpc"
	"google.golang.org/grpc"
)

type GrpcServerInfoClient struct {
	serverAddr string
	conn       *grpc.ClientConn
	client     pb.SvcInfoClient
}

func NewGrpcServerInfoClient(host string) (*GrpcServerInfoClient, error) {
	conn, err := Dial(host)
	if err != nil {
		return nil, err
	}
	client := pb.NewSvcInfoClient(conn)
	return &GrpcServerInfoClient{
		serverAddr: host,
		conn:       conn,
		client:     client,
	}, nil
}

func (c *GrpcServerInfoClient) Close() {
	c.conn.Close()
}

func (c *GrpcServerInfoClient) Info(ctx context.Context, in *pb.InfoRequest) ([]*pb.Info, error) {
	r, err := c.client.Info(ctx, &pb.InfoRequest{})
	if err != nil {
		return nil, err
	}
	return r.Info, nil

}
