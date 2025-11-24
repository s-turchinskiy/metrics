package grpchandlers

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
	"github.com/s-turchinskiy/metrics/internal/utils/hashutil"
	proto "github.com/s-turchinskiy/metrics/models/grps"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) HashReadInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	if s.hashKey == "" {
		return handler(ctx, req)
	}

	var requestHexadecimalHash string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get("HashSHA256")
		if len(values) > 0 {
			requestHexadecimalHash = values[0]
		}
	}
	if len(requestHexadecimalHash) == 0 {
		return handler(ctx, req)
	}

	requestHash, err := hex.DecodeString(requestHexadecimalHash)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Error decode request hash")
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

	expectedHash := hashutil.СomputeSha256Hash(s.hashKey, bodyBytes)

	if !hmac.Equal(requestHash, expectedHash) {
		return nil, status.Error(codes.Unauthenticated, "Invalid request hash")
	}

	return handler(ctx, req)
}
