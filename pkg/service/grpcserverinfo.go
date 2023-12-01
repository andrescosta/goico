package service

import (
	context "context"

	info "github.com/andrescosta/goico/pkg/service/info/grpc"
)

type GrpcServerInfo struct {
	info.UnimplementedSvcInfoServer
	svc *GrpcService
}

func NewGrpcServerInfo(svc *GrpcService) *GrpcServerInfo {
	return &GrpcServerInfo{
		svc: svc,
	}
}

func (g *GrpcServerInfo) Info(ctx context.Context, req *info.InfoRequest) (*info.InfoResponse, error) {
	i := make([]*info.Info, 0)
	for k, v := range g.svc.Info() {
		i = append(i, &info.Info{
			Key:   k,
			Value: v,
		})
	}

	return &info.InfoResponse{Info: i}, nil
}
