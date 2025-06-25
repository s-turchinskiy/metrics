package main

import (
	"errors"
	"fmt"
	"strconv"
)

type MetricsStorage struct {
	Gauge   map[string]float64
	Counter map[string]int64
}

func (s *MetricsStorage) UpdateMetric(metric Metric) error {

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

func (s *MetricsStorage) GetMetric(metric Metric) (string, error) {

	switch metricsType := metric.MetricsType; metricsType {
	case "gauge":

		value, exist := s.Gauge[metric.MetricsName]

		if !exist {
			return "0", nil
		}

		return fmt.Sprintf("%f", value), nil
	case "counter":

		value, exist := s.Counter[metric.MetricsName]

		if !exist {
			return "0", nil
		}

		return strconv.FormatInt(value, 10), nil

	default:
		return "", errMetricsTypeNotFound
	}
}

var (
	errMetricsTypeNotFound = errors.New("metrics type not found")
)
