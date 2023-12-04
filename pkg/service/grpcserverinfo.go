package service

import (
	context "context"

	"github.com/andrescosta/goico/pkg/service/svcmeta"
)

type GrpcServerInfo struct {
	svcmeta.UnimplementedGrpcMetadataServer
	svc *GrpcService
}

func NewGrpcServerInfo(svc *GrpcService) *GrpcServerInfo {
	return &GrpcServerInfo{
		svc: svc,
	}
}

func (g *GrpcServerInfo) Metadata(ctx context.Context, req *svcmeta.GrpcMetadataRequest) (*svcmeta.GrpcMetadataReply, error) {
	i := make([]*svcmeta.GrpcServerMetadata, 0)
	for k, v := range g.svc.Info() {
		i = append(i, &svcmeta.GrpcServerMetadata{
			Key:   k,
			Value: v,
		})
	}

	return &svcmeta.GrpcMetadataReply{Metadata: i}, nil
}
