package main

import (
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"
)

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
