package service

import (
	"context"

	"github.com/andrescosta/goico/pkg/service/svcmeta"
	"google.golang.org/grpc"
)

type GrpcServerInfoClient struct {
	serverAddr string
	conn       *grpc.ClientConn
	client     svcmeta.GrpcMetadataClient
}

func NewGrpcServerInfoClient(host string) (*GrpcServerInfoClient, error) {
	conn, err := Dial(host)
	if err != nil {
		return nil, err
	}
	client := svcmeta.NewGrpcMetadataClient(conn)
	return &GrpcServerInfoClient{
		serverAddr: host,
		conn:       conn,
		client:     client,
	}, nil
}

func (c *GrpcServerInfoClient) Close() {
	c.conn.Close()
}

func (c *GrpcServerInfoClient) Info(ctx context.Context, in *svcmeta.GrpcMetadataRequest) ([]*svcmeta.GrpcServerMetadata, error) {
	r, err := c.client.Metadata(ctx, in)
	if err != nil {
		return nil, err
	}
	return r.Metadata, nil

}
