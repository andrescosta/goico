package service

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	ctx, done := context.WithTimeout(ctx, 5*time.Second)
	c, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	done()
	if err != nil {
		return nil, err
	}
	return c, err
}
