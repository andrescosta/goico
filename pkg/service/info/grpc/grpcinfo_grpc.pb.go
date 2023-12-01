// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.25.1
// source: grpcinfo.proto

package info

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	SvcInfo_Info_FullMethodName = "/SvcInfo/Info"
)

// SvcInfoClient is the client API for SvcInfo service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SvcInfoClient interface {
	Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error)
}

type svcInfoClient struct {
	cc grpc.ClientConnInterface
}

func NewSvcInfoClient(cc grpc.ClientConnInterface) SvcInfoClient {
	return &svcInfoClient{cc}
}

func (c *svcInfoClient) Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error) {
	out := new(InfoResponse)
	err := c.cc.Invoke(ctx, SvcInfo_Info_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SvcInfoServer is the server API for SvcInfo service.
// All implementations must embed UnimplementedSvcInfoServer
// for forward compatibility
type SvcInfoServer interface {
	Info(context.Context, *InfoRequest) (*InfoResponse, error)
	mustEmbedUnimplementedSvcInfoServer()
}

// UnimplementedSvcInfoServer must be embedded to have forward compatible implementations.
type UnimplementedSvcInfoServer struct {
}

func (UnimplementedSvcInfoServer) Info(context.Context, *InfoRequest) (*InfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Info not implemented")
}
func (UnimplementedSvcInfoServer) mustEmbedUnimplementedSvcInfoServer() {}

// UnsafeSvcInfoServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SvcInfoServer will
// result in compilation errors.
type UnsafeSvcInfoServer interface {
	mustEmbedUnimplementedSvcInfoServer()
}

func RegisterSvcInfoServer(s grpc.ServiceRegistrar, srv SvcInfoServer) {
	s.RegisterService(&SvcInfo_ServiceDesc, srv)
}

func _SvcInfo_Info_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SvcInfoServer).Info(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SvcInfo_Info_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SvcInfoServer).Info(ctx, req.(*InfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// SvcInfo_ServiceDesc is the grpc.ServiceDesc for SvcInfo service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SvcInfo_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "SvcInfo",
	HandlerType: (*SvcInfoServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Info",
			Handler:    _SvcInfo_Info_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "grpcinfo.proto",
}
