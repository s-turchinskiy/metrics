package grpchandlers

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	proto "github.com/s-turchinskiy/metrics/models/grps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) AddMetric(ctx context.Context, request *proto.AddMetricRequest) (*proto.AddMetricResponse, error) {

	metric := request.Metric.GetStorageMetric()
	result, err := s.service.UpdateTypedMetric(ctx, metric)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if result.Delta == nil {
		result.Delta = new(int64)
	}

	if result.Value == nil {
		result.Value = new(float64)
	}

	m := proto.Metric{
		Id:    result.Name,
		Value: *result.Value,
		Delta: *result.Delta,
	}

	response := &proto.AddMetricResponse{
		Metric: m.SetMTypeFromString(result.MType),
	}
	return response, nil

}

func (s *GRPCServer) AddMetricBatch(ctx context.Context, request *proto.AddMetricsRequest) (*proto.AddMetricsResponse, error) {

	typedMetrics := make([]models.StorageMetrics, len(request.Metrics))
	for i, metric := range request.Metrics {
		typedMetrics[i] = metric.GetStorageMetric()
	}

	result, err := s.service.UpdateTypedMetrics(ctx, typedMetrics)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &proto.AddMetricsResponse{Count: result}, nil
}
