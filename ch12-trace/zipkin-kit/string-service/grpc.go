package main

import (
	"context"
	"github.com/gauge2009/micro-golang/ch12-trace/zipkin-kit/client"
	"github.com/gauge2009/micro-golang/ch12-trace/zipkin-kit/pb"
	"github.com/gauge2009/micro-golang/ch12-trace/zipkin-kit/string-service/endpoint"
	"github.com/go-kit/kit/transport/grpc"
)

type grpcServer struct {
	diff grpc.Handler
}

func (s *grpcServer) Diff(ctx context.Context, r *pb.StringRequest) (*pb.StringResponse, error) {
	_, resp, err := s.diff.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.StringResponse), nil

}

func NewGRPCServer(ctx context.Context, endpoints endpoint.StringEndpoints, serverTracer grpc.ServerOption) pb.StringServiceServer {
	return &grpcServer{
		diff: grpc.NewServer(
			endpoints.StringEndpoint,
			client.DecodeGRPCStringRequest,
			client.EncodeGRPCStringResponse,
			serverTracer,
		),
	}
}
