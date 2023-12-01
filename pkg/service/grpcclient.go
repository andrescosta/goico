package service

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Dial(addr string) (*grpc.ClientConn, error) {
	ops := grpc.WithTransportCredentials(insecure.NewCredentials())
	return grpc.Dial(addr, ops)
}
