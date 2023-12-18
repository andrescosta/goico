package grpcutil

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	if addr == "" {
		return nil, errors.New("service.Dial: address is empty, check env files")
	}
	c, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}
