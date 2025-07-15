package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"os"
	"strconv"
	"sync"
	"time"
)

type MetricsUpdater interface {
	UpdateMetric(metric models.UntypedMetric) error
	UpdateTypedMetric(metric models.StorageMetrics) (*models.StorageMetrics, error)
	GetMetric(metric models.UntypedMetric) (string, error)
	GetTypedMetric(metric models.StorageMetrics) (*models.StorageMetrics, error)
	GetAllMetrics() (map[string]map[string]string, error)
	SaveMetricsToFile() error
	LoadMetricsFromFile() error
	Ping() ([]byte, error)
}

type MetricsStorage struct {
	Repository Repository
	file       *os.File
	mutex      sync.Mutex
}

type Repository interface {
	UpdateGauge(metricsName string, newValue float64) error
	UpdateCounter(metricsName string, newValue int64) error
	CountGauges() int
	CountCounters() int
	GetGauge(metricsName string) (float64, bool, error)
	GetCounter(metricsName string) (int64, bool, error)
	GetAllGauges() (map[string]float64, error)
	GetAllCounters() (map[string]int64, error)
	ReloadAllGauges(map[string]float64) error
	ReloadAllCounters(map[string]int64) error
	Ping() ([]byte, error)
}

type MetricsFileStorage struct {
	Gauge   map[string]float64
	Counter map[string]int64
	Date    string
}

func (s *MetricsStorage) GetAllMetrics() (map[string]map[string]string, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	result := make(map[string]map[string]string, 2)

	gauges, err := s.Repository.GetAllGauges()
	if err != nil {
		return nil, err
	}
	resultGauges := make(map[string]string, len(gauges))

	for name, value := range gauges {
		resultGauges[name] = strconv.FormatFloat(value, 'f', -1, 64)
	}
	result["Gauge"] = resultGauges

	counters, err := s.Repository.GetAllCounters()
	if err != nil {
		return nil, err
	}
	resultCounters := make(map[string]string, len(counters))
	for name, value := range counters {
		resultCounters[name] = strconv.FormatInt(value, 10)
	}
	result["Counter"] = resultCounters

	return result, nil
}

func (s *MetricsStorage) UpdateTypedMetric(metric models.StorageMetrics) (*models.StorageMetrics, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	result := models.StorageMetrics{Name: metric.Name, MType: metric.MType}
	switch metricsType := metric.MType; metricsType {
	case "gauge":

		if metric.Value == nil {
			return &result, fmt.Errorf("value is not defined")
		}

		newValue := *metric.Value
		err := s.Repository.UpdateGauge(metric.Name, newValue)
		if err != nil {
			return nil, err
		}
		result.Value = &newValue
	case "counter":

		if metric.Delta == nil {
			return &result, fmt.Errorf("delta is not defined")
		}
		currentValue, exist, err := s.Repository.GetCounter(metric.Name)
		if err != nil {
			return nil, err
		}

		if !exist {
			newValue := *metric.Delta
			err := s.Repository.UpdateCounter(metric.Name, newValue)
			if err != nil {
				return nil, err
			}
			result.Delta = &newValue
			return &result, nil
		}

		newValue := currentValue + *metric.Delta
		err = s.Repository.UpdateCounter(metric.Name, newValue)
		if err != nil {
			return nil, err
		}
		result.Delta = &newValue

	default:
		return nil, errMetricsTypeNotFound
	}

	return &result, nil

}
func (s *MetricsStorage) UpdateMetric(metric models.UntypedMetric) error {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch metricsType := metric.MetricsType; metricsType {
	case "gauge":

		value, err := strconv.ParseFloat(metric.MetricsValue, 64)
		if err != nil {
			return fmt.Errorf("MetricsValue = %s, error: "+err.Error(), metric.MetricsValue)
		}

		err = s.Repository.UpdateGauge(metric.MetricsName, value)
		if err != nil {
			return err
		}

	case "counter":

		value, err := strconv.ParseInt(metric.MetricsValue, 10, 64)
		if err != nil {
			return err
		}

		currentValue, exist, err := s.Repository.GetCounter(metric.MetricsName)
		if err != nil {
			return err
		}

		if !exist {
			err = s.Repository.UpdateCounter(metric.MetricsName, value)
			return err
		}

		err = s.Repository.UpdateCounter(metric.MetricsName, currentValue+value)
		return err

	default:
		return errMetricsTypeNotFound
	}

	return nil
}

func (s *MetricsStorage) GetTypedMetric(metric models.StorageMetrics) (*models.StorageMetrics, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	result := models.StorageMetrics{Name: metric.Name, MType: metric.MType}

	switch metricsType := metric.MType; metricsType {
	case "gauge":

		value, exist, err := s.Repository.GetGauge(metric.Name)
		if err != nil {
			return nil, err
		}

		if !exist {
			var zero float64
			result.Value = &zero
			return &result, nil
		}

		result.Value = &value
		return &result, nil

	case "counter":

		value, exist, err := s.Repository.GetCounter(metric.Name)
		if err != nil {
			return nil, err
		}

		if !exist {
			var zero int64
			result.Delta = &zero
			return &result, nil
		}

		result.Delta = &value
		return &result, nil

	default:
		return nil, errMetricsTypeNotFound
	}
}

func (s *MetricsStorage) GetMetric(metric models.UntypedMetric) (string, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch metricsType := metric.MetricsType; metricsType {
	case "gauge":

		value, exist, err := s.Repository.GetGauge(metric.MetricsName)
		if err != nil {
			return "", err
		}

		if !exist {
			return "", fmt.Errorf("not found")
		}

		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case "counter":

		value, exist, err := s.Repository.GetCounter(metric.MetricsName)
		if err != nil {
			return "", err
		}

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
	defer s.mutex.Unlock()

	if s.Repository.CountGauges() == 0 && s.Repository.CountCounters() == 0 {
		logger.Log.Debug("SaveMetricsToFile, no data available")
		return nil
	}

	gauges, err := s.Repository.GetAllGauges()
	if err != nil {
		return err
	}

	counters, err := s.Repository.GetAllCounters()
	if err != nil {
		return err
	}

	metricsForFile := MetricsFileStorage{
		Gauge:   gauges,
		Counter: counters,
		Date:    time.Now().Format(time.DateTime),
	}

	data, err := json.MarshalIndent(&metricsForFile, "", "   ")
	if err != nil {
		return err
	}

	err = os.WriteFile(settings.Settings.FileStoragePath, data, 0666)
	if err != nil {
		return err
	}

	logger.Log.Debugw("SaveMetricsToFile", "data", string(data))

	return err

}

func (s *MetricsStorage) LoadMetricsFromFile() error {

	metricsForFile := &MetricsFileStorage{}

	data, err := os.ReadFile(settings.Settings.FileStoragePath)

	if errors.Is(err, os.ErrNotExist) {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		logger.Log.Debug(fmt.Sprintf("file %s%s not exist", dir, settings.Settings.FileStoragePath))
		return nil

	}
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, metricsForFile); err != nil {
		return err
	}

	s.mutex.Lock()
	err = s.Repository.ReloadAllGauges(metricsForFile.Gauge)
	if err != nil {
		return err
	}
	err = s.Repository.ReloadAllCounters(metricsForFile.Counter)
	if err != nil {
		return err
	}
	s.mutex.Unlock()

	logger.Log.Debugw("LoadMetricsFromFile", "data", string(data))

	return err

}

func (s *MetricsStorage) Ping() ([]byte, error) {

	return s.Repository.Ping()

}

var (
	errMetricsTypeNotFound = errors.New("metrics type not found")
)
