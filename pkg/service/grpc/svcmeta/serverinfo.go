package svcmeta

import (
	context "context"
)

type ServerInfo struct {
	UnimplementedGrpcMetadataServer
	metadata map[string]string
}

func NewServerInfo(metadata map[string]string) *ServerInfo {
	return &ServerInfo{
		metadata: metadata,
	}
}

func (g *ServerInfo) Metadata(_ context.Context, _ *GrpcMetadataRequest) (*GrpcMetadataReply, error) {
	i := make([]*GrpcServerMetadata, 0)
	for k, v := range g.metadata {
		i = append(i, &GrpcServerMetadata{
			Key:   k,
			Value: v,
		})
	}
	return &GrpcMetadataReply{Metadata: i}, nil
}
