package grpc

import (
	"context"

	"github.com/andrescosta/goico/pkg/service/svcmeta"
	"google.golang.org/grpc"
)

type ServerInfoClient struct {
	serverAddr string

	conn *grpc.ClientConn

	client svcmeta.GrpcMetadataClient
}

func NewInfoClient(ctx context.Context, host string) (*ServerInfoClient, error) {
	conn, err := Dial(ctx, host)

	if err != nil {
		return nil, err
	}

	client := svcmeta.NewGrpcMetadataClient(conn)

	return &ServerInfoClient{

		serverAddr: host,

		conn: conn,

		client: client,
	}, nil
}

func (c *ServerInfoClient) Close() {
	c.conn.Close()
}

func (c *ServerInfoClient) Info(ctx context.Context, in *svcmeta.GrpcMetadataRequest) ([]*svcmeta.GrpcServerMetadata, error) {
	r, err := c.client.Metadata(ctx, in)

	if err != nil {
		return nil, err
	}

	return r.Metadata, nil
}
