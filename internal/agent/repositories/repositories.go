// Package repositories Хранение метрик
package repositories

import (
	"sync"

	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
)

type MetricsRepositorier interface {
	UpdateMetrics(map[string]float64) error
	GetMetrics() ([]models.Metrics, error)
}

type MetricsStorage struct {
	Gauge   map[string]float64
	Counter map[string]int64
	mutex   sync.Mutex
}

func (s *MetricsStorage) GetMetrics() ([]models.Metrics, error) {

	var result []models.Metrics

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for ID, value := range s.Gauge {

		metric := models.Metrics{ID: ID, MType: "gauge", Value: &value}
		result = append(result, metric)
	}

	for ID, value := range s.Counter {

		metric := models.Metrics{ID: ID, MType: "counter", Delta: &value}
		result = append(result, metric)
	}

	logger.Log.Debugw("GetMetrics", "PollCount", s.Counter["PollCount"])

	return result, nil

}

func (s *MetricsStorage) UpdateMetrics(metrics map[string]float64) error {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Gauge = metrics
	s.Counter["PollCount"]++

	logger.Log.Debugw("UpdateMetrics", "PollCount", s.Counter["PollCount"])

	return nil
}
