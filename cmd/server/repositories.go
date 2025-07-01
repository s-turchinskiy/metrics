package main

import (
	"errors"
	"fmt"
	"github.com/s-turchinskiy/metrics/cmd/server/internal/models"
	"strconv"
	"unicode/utf8"
)

type MetricsUpdater interface {
	UpdateMetric(metric models.UntypedMetric) error
	UpdateTypedMetric(metric models.StorageMetrics) (models.StorageMetrics, error)
	GetMetric(metric models.UntypedMetric) (string, error)
	GetTypedMetric(metric models.StorageMetrics) (models.StorageMetrics, error)
	GetAllMetrics() ([]string, error)
}

type MetricsStorage struct {
	Gauge   map[string]float64
	Counter map[string]int64
}

func (s *MetricsStorage) GetAllMetrics() ([]string, error) {

	var result []string
	result = append(result, "Gauge")
	for name, value := range s.Gauge {

		var extraIndent string
		if utf8.RuneCountInString(name) <= 6 {
			extraIndent = "\t"
		}
		result = append(result, fmt.Sprintf("\t%s:%s\t%s", name, extraIndent, strconv.FormatFloat(value, 'f', -1, 64)))
	}

	result = append(result, "Counter")
	for name, value := range s.Counter {
		var extraIndent string
		if utf8.RuneCountInString(name) <= 6 {
			extraIndent = "\t"
		}
		result = append(result, fmt.Sprintf("\t%s:%s\t%s", name, extraIndent, strconv.FormatInt(value, 10)))
	}
	return result, nil
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
			return result, fmt.Errorf("not found")
		}

		result.Value = &value
		return result, nil
	case "counter":

		value, exist := s.Counter[metric.Name]

		if !exist {
			return result, fmt.Errorf("not found")
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
