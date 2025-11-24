package grpchandlers

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
)

func (s *GRPCServer) TrustedSubnetInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	if s.trustedSubnet == nil {
		return handler(ctx, req)
	}

	var realIP string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get("X-Real-IP")
		if len(values) > 0 {
			realIP = values[0]
		}
	}
	if len(realIP) == 0 {
		return nil, status.Error(codes.PermissionDenied, "empty Header X-Real-IP")
	}

	ip := net.ParseIP(realIP)
	if ip == nil {
		return nil, status.Error(codes.PermissionDenied, "Invalid IP address in Header X-Real-IP")
	}

	if !s.trustedSubnet.Contains(ip) {

		logger.Log.Infow("IP address not in allowed in this subnet",
			zap.String("ip", realIP),
			zap.String("subnet", s.trustedSubnet.String()))

		return nil, status.Error(codes.PermissionDenied, "IP address not in allowed in this subnet")

	}

	return handler(ctx, req)
}
