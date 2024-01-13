package svcmeta

import (
	"context"

	"github.com/andrescosta/goico/pkg/service/grpc/grpcutil"
	rpc "google.golang.org/grpc"
)

type InfoClient struct {
	serverAddr string
	conn       *rpc.ClientConn
	client     GrpcMetadataClient
}

func NewInfoClient(ctx context.Context, target string) (*InfoClient, error) {
	conn, err := grpcutil.Dial(ctx, target)
	if err != nil {
		return nil, err
	}
	return NewInfoClientWithConn(conn)
}

func NewInfoClientWithConn(conn *rpc.ClientConn) (*InfoClient, error) {
	client := NewGrpcMetadataClient(conn)
	return &InfoClient{
		serverAddr: conn.Target(),
		conn:       conn,
		client:     client,
	}, nil
}

func (c *InfoClient) Close() {
	_ = c.conn.Close()
}

func (c *InfoClient) Info(ctx context.Context, in *GrpcMetadataRequest) ([]*GrpcServerMetadata, error) {
	r, err := c.client.Metadata(ctx, in)
	if err != nil {
		return nil, err
	}
	return r.Metadata, nil
}

func (c *InfoClient) InfoAsMap(ctx context.Context, in *GrpcMetadataRequest) (map[string]string, error) {
	i, err := c.Info(ctx, in)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(i))
	for _, ii := range i {
		m[ii.Key] = ii.Value
	}
	return m, nil
}
