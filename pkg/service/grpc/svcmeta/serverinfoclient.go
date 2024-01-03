package svcmeta

import (
	"context"

	"github.com/andrescosta/goico/pkg/service/grpc/grpcutil"
	rpc "google.golang.org/grpc"
)

type ServerInfoClient struct {
	serverAddr string
	conn       *rpc.ClientConn
	client     GrpcMetadataClient
}

func NewInfoClient(ctx context.Context, host string) (*ServerInfoClient, error) {
	conn, err := grpcutil.Dial(ctx, host)
	if err != nil {
		return nil, err
	}
	client := NewGrpcMetadataClient(conn)
	return &ServerInfoClient{
		serverAddr: host,
		conn:       conn,
		client:     client,
	}, nil
}

func (c *ServerInfoClient) Close() {
	_ = c.conn.Close()
}

func (c *ServerInfoClient) Info(ctx context.Context, in *GrpcMetadataRequest) ([]*GrpcServerMetadata, error) {
	r, err := c.client.Metadata(ctx, in)
	if err != nil {
		return nil, err
	}
	return r.Metadata, nil
}
