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

	switch info.FullMethod {
	case pathAddMetric:
		typedReq := req.(*proto.AddMetricRequest)
		bodyBytes = typedReq.Body
	case pathAddMetrics:
		typedReq := req.(*proto.AddMetricsRequest)
		bodyBytes = typedReq.Body
	default:
		return handler(ctx, req)
	}

	if len(bodyBytes) == 0 {
		return handler(ctx, req)
	}

	bodyBytes, err := rsautil.Decrypt(s.privateKey, bodyBytes)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "cannot decrypt body")
	}

	switch info.FullMethod {
	case pathAddMetric:
		typedReq := req.(*proto.AddMetricRequest)
		typedReq.Body = bodyBytes
		req = typedReq
	case pathAddMetrics:
		typedReq := req.(*proto.AddMetricsRequest)
		typedReq.Body = bodyBytes
		req = typedReq
	default:
		return handler(ctx, req)
	}

	return handler(ctx, req)
}
