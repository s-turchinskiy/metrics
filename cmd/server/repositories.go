package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"os"
	"strconv"
	"sync"
	"time"
)

type MetricsUpdater interface {
	UpdateMetric(metric models.UntypedMetric) error
	UpdateTypedMetric(metric models.StorageMetrics) (models.StorageMetrics, error)
	GetMetric(metric models.UntypedMetric) (string, error)
	GetTypedMetric(metric models.StorageMetrics) (models.StorageMetrics, error)
	GetAllMetrics() map[string]map[string]string
	SaveMetricsToFile() error
	LoadMetricsFromFile() error
}

type MetricsStorage struct {
	Gauge   map[string]float64
	Counter map[string]int64
	file    *os.File
	mutex   sync.Mutex
}

type MetricsForFile struct {
	Metrics *MetricsStorage
	Date    string
}

func (s *MetricsStorage) GetAllMetrics() map[string]map[string]string {

	s.mutex.Lock()

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

	s.mutex.Unlock()
	return result
}

func (s *MetricsStorage) UpdateTypedMetric(metric models.StorageMetrics) (models.StorageMetrics, error) {

	result := models.StorageMetrics{Name: metric.Name, MType: metric.MType}
	switch metricsType := metric.MType; metricsType {
	case "gauge":

		newValue := *metric.Value
		s.mutex.Lock()
		s.Gauge[metric.Name] = newValue
		s.mutex.Unlock()
		result.Value = &newValue
	case "counter":

		s.mutex.Lock()
		currentValue, exist := s.Counter[metric.Name]

		if !exist {
			s.Counter[metric.Name] = *metric.Delta
			s.mutex.Unlock()
			return result, nil
		}

		newValue := currentValue + *metric.Delta
		s.Counter[metric.Name] = newValue
		s.mutex.Unlock()
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

		s.mutex.Lock()
		s.Gauge[metric.MetricsName] = value
		s.mutex.Unlock()

	case "counter":

		value, err := strconv.ParseInt(metric.MetricsValue, 10, 64)
		if err != nil {
			return err
		}

		s.mutex.Lock()
		currentValue, exist := s.Counter[metric.MetricsName]

		if !exist {
			s.Counter[metric.MetricsName] = value
			s.mutex.Unlock()
			return nil
		}

		s.Counter[metric.MetricsName] = currentValue + value
		s.mutex.Unlock()

	default:
		return errMetricsTypeNotFound
	}

	return nil
}

func (s *MetricsStorage) GetTypedMetric(metric models.StorageMetrics) (models.StorageMetrics, error) {

	result := models.StorageMetrics{Name: metric.Name, MType: metric.MType}

	switch metricsType := metric.MType; metricsType {
	case "gauge":

		s.mutex.Lock()
		value, exist := s.Gauge[metric.Name]
		s.mutex.Unlock()

		if !exist {
			var zero float64
			result.Value = &zero
			return result, nil
		}

		result.Value = &value
		return result, nil

	case "counter":

		s.mutex.Lock()
		value, exist := s.Counter[metric.Name]
		s.mutex.Unlock()

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

		s.mutex.Lock()
		value, exist := s.Gauge[metric.MetricsName]
		s.mutex.Unlock()

		if !exist {
			return "", fmt.Errorf("not found")
		}

		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case "counter":

		s.mutex.Lock()
		value, exist := s.Counter[metric.MetricsName]
		s.mutex.Unlock()

		if !exist {
			return "", fmt.Errorf("not found")
		}

		return strconv.FormatInt(value, 10), nil

	default:
		return "", errMetricsTypeNotFound
	}
}

func (s *MetricsStorage) SaveMetricsToFile() error {

	s.mutex.Lock()

	if len(s.Gauge) == 0 && len(s.Counter) == 0 {
		s.mutex.Unlock()
		logger.Log.Debug("SaveMetricsToFile, no data available")
		return nil
	}
	metricsForFile := MetricsForFile{Metrics: s, Date: time.Now().Format(time.DateTime)}

	data, err := json.MarshalIndent(&metricsForFile, "", "   ")
	if err != nil {
		s.mutex.Unlock()
		return err
	}

	s.mutex.Unlock()

	err = os.WriteFile(settings.FileStoragePath, data, 0666)
	if err != nil {
		return err
	}

	logger.Log.Debugw("SaveMetricsToFile", "data", string(data))

	return err

}

func (s *MetricsStorage) LoadMetricsFromFile() error {

	metricsForFile := &MetricsForFile{}

	/*file, err := os.OpenFile(settings.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(metricsForFile)
	if err != nil {
		return err
	}*/

	data, err := os.ReadFile(settings.FileStoragePath)

	if errors.Is(err, os.ErrNotExist) {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		logger.Log.Debug(fmt.Sprintf("file %s%s not exist", dir, settings.FileStoragePath))
		return nil

	}
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, metricsForFile); err != nil {
		return err
	}

	s.mutex.Lock()
	s.Gauge = metricsForFile.Metrics.Gauge
	s.Counter = metricsForFile.Metrics.Counter
	s.mutex.Unlock()

	logger.Log.Debugw("LoadMetricsFromFile", "data", string(data))

	return err

}

var (
	errMetricsTypeNotFound = errors.New("metrics type not found")
)
