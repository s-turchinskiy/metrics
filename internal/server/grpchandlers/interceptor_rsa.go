package grpchandlers

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/utils/rsautil"
	proto "github.com/s-turchinskiy/metrics/models/grps"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) RSADecryptInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	if s.privateKey == nil {
		return handler(ctx, req)
	}

	var bodyBytes []byte
	if info.FullMethod == "/protofile.Metrics/AddMetric" {
		typedReq := req.(*proto.AddMetricRequest)
		bodyBytes = typedReq.Metric.Body
	} else {
		return handler(ctx, req)
	}

	if len(bodyBytes) == 0 {
		return handler(ctx, req)
	}

	bodyBytes, err := rsautil.Decrypt(s.privateKey, bodyBytes)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "cannot decrypt body")
	}

	if info.FullMethod == "/protofile.Metrics/AddMetric" {
		typedReq := req.(*proto.AddMetricRequest)
		typedReq.Metric.Body = bodyBytes
		req = typedReq
	}

	return handler(ctx, req)
}
