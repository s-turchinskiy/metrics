package service

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/server/models"
)

type MetricsUpdater interface {
	UpdateMetric(ctx context.Context, metric models.UntypedMetric) error
	UpdateTypedMetric(ctx context.Context, metric models.StorageMetrics) (*models.StorageMetrics, error)
	UpdateTypedMetrics(ctx context.Context, metric []models.StorageMetrics) (int64, error)
	GetMetric(ctx context.Context, metric models.UntypedMetric) (string, error)
	GetTypedMetric(ctx context.Context, metric models.StorageMetrics) (*models.StorageMetrics, error)
	GetAllMetrics(ctx context.Context) (map[string]map[string]string, error)
	SaveMetricsToFile(ctx context.Context) error
	LoadMetricsFromFile(ctx context.Context) error
	Ping(ctx context.Context) ([]byte, error)
}
