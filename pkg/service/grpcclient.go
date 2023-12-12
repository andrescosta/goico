package service

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrAddress = errors.New("address is empty, check env files")
)

func Dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	if addr == "" {
		return nil, ErrAddress
	}
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
