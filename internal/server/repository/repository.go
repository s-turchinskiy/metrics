package repository

import (
	"context"

	"github.com/s-turchinskiy/metrics/internal/server/models"
)

type Repository interface {
	UpdateGauge(ctx context.Context, metricsName string, newValue float64) error
	UpdateCounter(ctx context.Context, metricsName string, newValue int64) error
	CountGauges(ctx context.Context) int
	CountCounters(ctx context.Context) int
	GetGauge(ctx context.Context, metricsName string) (float64, bool, error)
	GetCounter(ctx context.Context, metricsName string) (int64, bool, error)
	GetAllGauges(ctx context.Context) (map[string]float64, error)
	GetAllCounters(ctx context.Context) (map[string]int64, error)
	ReloadAllGauges(context.Context, map[string]float64) error
	ReloadAllCounters(context.Context, map[string]int64) error
	ReloadAllMetrics(context.Context, []models.StorageMetrics) (int64, error)

	Ping(ctx context.Context) ([]byte, error)
}
