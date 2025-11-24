package grpchandlers

import (
	"context"
	"crypto/rsa"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/service"
	proto "github.com/s-turchinskiy/metrics/models/grps"
	"google.golang.org/grpc"
	"log"
	"net"
)

const (
	pathAddMetric  = "/protofile.Metrics/AddMetric"
	pathAddMetrics = "/protofile.Metrics/AddMetrics"
)

type GRPCServer struct {
	proto.UnimplementedMetricsServer
	service       service.MetricsUpdater
	listen        net.Listener
	server        *grpc.Server
	hashKey       string
	privateKey    *rsa.PrivateKey
	trustedSubnet *net.IPNet
}

func New(service service.MetricsUpdater, port, hashKey string, privateKey *rsa.PrivateKey, trustedSubnet *net.IPNet) *GRPCServer {

	s := &GRPCServer{
		service:       service,
		hashKey:       hashKey,
		privateKey:    privateKey,
		trustedSubnet: trustedSubnet,
	}

	var err error
	s.listen, err = net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	s.server = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			s.TrustedSubnetInterceptor,
			s.RSADecryptInterceptor,
			s.HashReadInterceptor),
	)

	proto.RegisterMetricsServer(s.server, s)

	logger.Log.Infow("Сервер gRPC начал работу")

	return s

}

func (s *GRPCServer) Run() error {

	return s.server.Serve(s.listen)
}

func (s *GRPCServer) Close(ctx context.Context) error {

	s.server.Stop()
	return nil

}
