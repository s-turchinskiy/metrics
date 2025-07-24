package memcashed

import (
	"context"
	"errors"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/server/models"
)

type MemCashed struct {
	Gauge   map[string]float64
	Counter map[string]int64
}

func (m *MemCashed) ReloadAllMetrics(ctx context.Context, metrics []models.StorageMetrics) (int64, error) {

	m.Gauge = make(map[string]float64)
	m.Counter = make(map[string]int64)

	var errs []error

	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			err := m.UpdateGauge(ctx, metric.Name, *metric.Value)
			if err != nil {
				errs = append(errs, err)
			}
		case "counter":
			err := m.UpdateCounter(ctx, metric.Name, *metric.Delta)
			if err != nil {
				errs = append(errs, err)
			}
		default:
			return 0, fmt.Errorf("unclown MType %s", metric.MType)

		}

	}

	return int64(len(m.Gauge) + len(m.Counter)), errors.Join(errs...)
}

func (m *MemCashed) Ping(ctx context.Context) ([]byte, error) {
	return nil, nil
}

func (m *MemCashed) ReloadAllGauges(ctx context.Context, newValue map[string]float64) error {
	m.Gauge = newValue
	return nil
}

func (m *MemCashed) ReloadAllCounters(ctx context.Context, newValue map[string]int64) error {
	m.Counter = newValue
	return nil
}

func (m *MemCashed) GetAllGauges(ctx context.Context) (map[string]float64, error) {

	return m.Gauge, nil
}

func (m *MemCashed) GetAllCounters(ctx context.Context) (map[string]int64, error) {

	return m.Counter, nil

}

func (m *MemCashed) GetGauge(ctx context.Context, metricsName string) (float64, bool, error) {
	v, exist := m.Gauge[metricsName]
	return v, exist, nil
}

func (m *MemCashed) GetCounter(ctx context.Context, metricsName string) (int64, bool, error) {
	v, exist := m.Counter[metricsName]
	return v, exist, nil
}

func (m *MemCashed) CountGauges(ctx context.Context) int {
	return len(m.Gauge)
}

func (m *MemCashed) CountCounters(ctx context.Context) int {
	return len(m.Counter)
}

func (m *MemCashed) UpdateCounter(ctx context.Context, metricsName string, delta int64) error {

	_, exist, err := m.GetCounter(ctx, metricsName)
	if err != nil {
		return err
	}

	if !exist {
		m.Counter[metricsName] = delta
	} else {
		m.Counter[metricsName] += delta
	}

	return nil

}

func (m *MemCashed) UpdateGauge(ctx context.Context, metricsName string, newValue float64) error {

	m.Gauge[metricsName] = newValue
	return nil

}
