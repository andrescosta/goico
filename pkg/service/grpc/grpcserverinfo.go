package grpc

import (
	context "context"

	"github.com/andrescosta/goico/pkg/service/svcmeta"
)

type ServerInfo struct {
	svcmeta.UnimplementedGrpcMetadataServer
	svc *Service
}

func NewServerInfo(svc *Service) *ServerInfo {
	return &ServerInfo{
		svc: svc,
	}
}

func (g *ServerInfo) Metadata(_ context.Context, _ *svcmeta.GrpcMetadataRequest) (*svcmeta.GrpcMetadataReply, error) {
	i := make([]*svcmeta.GrpcServerMetadata, 0)
	for k, v := range g.svc.Info() {
		i = append(i, &svcmeta.GrpcServerMetadata{
			Key:   k,
			Value: v,
		})
	}
	return &svcmeta.GrpcMetadataReply{Metadata: i}, nil
}
