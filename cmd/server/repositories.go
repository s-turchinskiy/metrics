package main

import (
	"errors"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"strconv"
)

type MetricsUpdater interface {
	UpdateMetric(metric models.UntypedMetric) error
	UpdateTypedMetric(metric models.StorageMetrics) (models.StorageMetrics, error)
	GetMetric(metric models.UntypedMetric) (string, error)
	GetTypedMetric(metric models.StorageMetrics) (models.StorageMetrics, error)
	GetAllMetrics() map[string]map[string]string
}

type MetricsStorage struct {
	Gauge   map[string]float64
	Counter map[string]int64
}

func (s *MetricsStorage) GetAllMetrics() map[string]map[string]string {

	result := make(map[string]map[string]string, 2)

	gauges := make(map[string]string, len(s.Gauge))
	for name, value := range s.Gauge {

		gauges[name] = strconv.FormatFloat(value, 'f', -1, 64)
	}
	result["Gauge"] = gauges

	counters := make(map[string]string, len(s.Gauge))
	for name, value := range s.Counter {
		counters[name] = strconv.FormatInt(value, 10)
	}
	result["Counter"] = counters

	return result
}

func (s *MetricsStorage) UpdateTypedMetric(metric models.StorageMetrics) (models.StorageMetrics, error) {

	result := models.StorageMetrics{Name: metric.Name, MType: metric.MType}
	switch metricsType := metric.MType; metricsType {
	case "gauge":

		newValue := *metric.Value
		s.Gauge[metric.Name] = newValue
		result.Value = &newValue
	case "counter":

		currentValue, exist := s.Counter[metric.Name]

		if !exist {
			s.Counter[metric.Name] = *metric.Delta
			return result, nil
		}

		newValue := currentValue + *metric.Delta
		s.Counter[metric.Name] = newValue
		result.Delta = &newValue

	default:
		return result, errMetricsTypeNotFound
	}

	return result, nil

}
func (s *MetricsStorage) UpdateMetric(metric models.UntypedMetric) error {

	switch metricsType := metric.MetricsType; metricsType {
	case "gauge":

		value, err := strconv.ParseFloat(metric.MetricsValue, 64)
		if err != nil {
			return fmt.Errorf("MetricsValue = %s, error: "+err.Error(), metric.MetricsValue)
		}

		s.Gauge[metric.MetricsName] = value
	case "counter":

		value, err := strconv.ParseInt(metric.MetricsValue, 10, 64)
		if err != nil {
			return err
		}

		currentValue, exist := s.Counter[metric.MetricsName]

		if !exist {
			s.Counter[metric.MetricsName] = value
			return nil
		}

		s.Counter[metric.MetricsName] = currentValue + value

	default:
		return errMetricsTypeNotFound
	}

	return nil
}

func (s *MetricsStorage) GetTypedMetric(metric models.StorageMetrics) (models.StorageMetrics, error) {

	result := models.StorageMetrics{Name: metric.Name, MType: metric.MType}

	switch metricsType := metric.MType; metricsType {
	case "gauge":

		value, exist := s.Gauge[metric.Name]

		if !exist {
			var zero float64
			result.Value = &zero
			return result, nil
		}

		result.Value = &value
		return result, nil
	case "counter":

		value, exist := s.Counter[metric.Name]

		if !exist {
			var zero int64
			result.Delta = &zero
			return result, nil
		}

		result.Delta = &value
		return result, nil

	default:
		return result, errMetricsTypeNotFound
	}
}

func (s *MetricsStorage) GetMetric(metric models.UntypedMetric) (string, error) {

	switch metricsType := metric.MetricsType; metricsType {
	case "gauge":

		value, exist := s.Gauge[metric.MetricsName]

		if !exist {
			return "", fmt.Errorf("not found")
		}

		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case "counter":

		value, exist := s.Counter[metric.MetricsName]

		if !exist {
			return "", fmt.Errorf("not found")
		}

		return strconv.FormatInt(value, 10), nil

	default:
		return "", errMetricsTypeNotFound
	}
}

var (
	errMetricsTypeNotFound = errors.New("metrics type not found")
)
