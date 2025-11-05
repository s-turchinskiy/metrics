package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/common/errutil"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"github.com/s-turchinskiy/metrics/internal/server/repository"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
)

type Service struct {
	Repository    repository.Repository
	retryStrategy []time.Duration
	mutex         sync.Mutex
}

// New Создание нового сервиса
func New(rep repository.Repository, retryStrategy []time.Duration) *Service {

	if len(retryStrategy) == 0 {
		retryStrategy = []time.Duration{0}
	}
	return &Service{
		Repository:    rep,
		retryStrategy: retryStrategy,
	}
}

type MetricsFileStorage struct {
	Gauge   map[string]float64
	Counter map[string]int64
	Date    string
}

// UpdateTypedMetrics Массовое обновление метрик
func (s *Service) UpdateTypedMetrics(ctx context.Context, metrics []models.StorageMetrics) (int64, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, delay := range s.retryStrategy {
		time.Sleep(delay)
		result, err := s.Repository.ReloadAllMetrics(ctx, metrics)
		if !isConnectionError(err) {
			return result, err
		}
	}

	return 0, errRetryStrategyIsNotDefined

}

// GetAllMetrics Получение всех метрик
func (s *Service) GetAllMetrics(ctx context.Context) (map[string]map[string]string, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	result := make(map[string]map[string]string, 2)

	var gauges map[string]float64
	var err error
	for _, delay := range s.retryStrategy {
		time.Sleep(delay)
		gauges, err = s.Repository.GetAllGauges(ctx)
		if err == nil {
			break
		} else if !isConnectionError(err) {
			return nil, err
		}
	}

	resultGauges := make(map[string]string, len(gauges))

	for name, value := range gauges {
		resultGauges[name] = strconv.FormatFloat(value, 'f', -1, 64)
	}
	result["Gauge"] = resultGauges

	var counters map[string]int64
	for _, delay := range s.retryStrategy {
		time.Sleep(delay)
		counters, err = s.Repository.GetAllCounters(ctx)
		if err == nil {
			break
		} else if !isConnectionError(err) {
			return nil, err
		}
	}

	resultCounters := make(map[string]string, len(counters))
	for name, value := range counters {
		resultCounters[name] = strconv.FormatInt(value, 10)
	}
	result["Counter"] = resultCounters

	return result, nil
}

// UpdateTypedMetric Обновление типизированной метрики
func (s *Service) UpdateTypedMetric(ctx context.Context, metric models.StorageMetrics) (*models.StorageMetrics, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	result := models.StorageMetrics{Name: metric.Name, MType: metric.MType}
	switch metricsType := metric.MType; metricsType {
	case "gauge":

		if metric.Value == nil {
			return &result, fmt.Errorf("value is not defined")
		}

		newValue := *metric.Value

		for _, delay := range s.retryStrategy {
			time.Sleep(delay)
			err := s.Repository.UpdateGauge(ctx, metric.Name, newValue)
			if err == nil {
				break
			} else if !isConnectionError(err) {
				return nil, err
			}
		}

		result.Value = &newValue
	case "counter":

		if metric.Delta == nil {
			return &result, fmt.Errorf("delta is not defined")
		}

		for _, delay := range s.retryStrategy {
			time.Sleep(delay)
			err := s.Repository.UpdateCounter(ctx, metric.Name, *metric.Delta)
			if err == nil {
				break
			} else if !isConnectionError(err) {
				return nil, err
			}
		}

		var value int64
		var err error
		for _, delay := range s.retryStrategy {
			time.Sleep(delay)
			value, _, err = s.Repository.GetCounter(ctx, metric.Name)
			if err == nil {
				break
			} else if !isConnectionError(err) {
				return nil, err
			}
		}

		result.Delta = &value

	default:
		return nil, errMetricsTypeNotFound
	}

	return &result, nil

}

// UpdateMetric Обновление нетипизированной метрики
func (s *Service) UpdateMetric(ctx context.Context, metric models.UntypedMetric) error {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch metricsType := metric.MetricsType; metricsType {
	case "gauge":

		value, err := strconv.ParseFloat(metric.MetricsValue, 64)
		if err != nil {
			return fmt.Errorf("MetricsValue = %s, error: "+err.Error(), metric.MetricsValue)
		}

		err = s.Repository.UpdateGauge(ctx, metric.MetricsName, value)
		if err != nil {
			return err
		}

	case "counter":

		delta, err := strconv.ParseInt(metric.MetricsValue, 10, 64)
		if err != nil {
			return err
		}

		err = s.Repository.UpdateCounter(ctx, metric.MetricsName, delta)
		if err != nil {
			return err
		}

		return err

	default:
		return errMetricsTypeNotFound
	}

	return nil
}

// GetTypedMetric Получение типизированной метрики
func (s *Service) GetTypedMetric(ctx context.Context, metric models.StorageMetrics) (*models.StorageMetrics, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	result := models.StorageMetrics{Name: metric.Name, MType: metric.MType}

	switch metricsType := metric.MType; metricsType {
	case "gauge":

		var value float64
		var exist bool
		var err error

		for _, delay := range s.retryStrategy {
			time.Sleep(delay)
			value, exist, err = s.Repository.GetGauge(ctx, metric.Name)
			if err == nil {
				break
			} else if !isConnectionError(err) {
				return nil, err
			}
		}

		if !exist {
			var zero float64
			result.Value = &zero
			return &result, nil
		}

		result.Value = &value
		return &result, nil

	case "counter":

		var value int64
		var exist bool
		var err error

		for _, delay := range s.retryStrategy {
			time.Sleep(delay)
			value, exist, err = s.Repository.GetCounter(ctx, metric.Name)
			if err == nil {
				break
			} else if !isConnectionError(err) {
				return nil, err
			}
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

// GetMetric Получение нетипизированной метрики
func (s *Service) GetMetric(ctx context.Context, metric models.UntypedMetric) (string, error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch metricsType := metric.MetricsType; metricsType {
	case "gauge":

		value, exist, err := s.Repository.GetGauge(ctx, metric.MetricsName)
		if err != nil {
			return "", err
		}

		if !exist {
			return "", fmt.Errorf("not found")
		}

		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case "counter":

		value, exist, err := s.Repository.GetCounter(ctx, metric.MetricsName)
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

// GetMetricsFromRepository Получение всех метрик для сохранения в файл
func (s *Service) GetMetricsFromRepository(ctx context.Context) (data []byte, err error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Repository.CountGauges(ctx) == 0 && s.Repository.CountCounters(ctx) == 0 {
		logger.Log.Debug("SaveMetricsToFile, no data available")
		return nil, nil
	}

	gauges, err := s.Repository.GetAllGauges(ctx)
	if err != nil {
		return nil, err
	}

	counters, err := s.Repository.GetAllCounters(ctx)
	if err != nil {
		return nil, err
	}

	metricsForFile := MetricsFileStorage{
		Gauge:   gauges,
		Counter: counters,
		Date:    time.Now().Format(time.DateTime),
	}

	return json.MarshalIndent(&metricsForFile, "", "   ")
}

// SaveMetricsToFile Сохранение метрик в файл
func (s *Service) SaveMetricsToFile(ctx context.Context) error {

	data, err := s.GetMetricsFromRepository(ctx)
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

// LoadMetricsFromData Загрузка метрик из массива байт
func (s *Service) LoadMetricsFromData(ctx context.Context, data []byte) error {

	metricsForFile := &MetricsFileStorage{}

	if err := json.Unmarshal(data, metricsForFile); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.Repository.ReloadAllGauges(ctx, metricsForFile.Gauge)
	if err != nil {
		return err
	}
	err = s.Repository.ReloadAllCounters(ctx, metricsForFile.Counter)
	if err != nil {
		return err
	}

	logger.Log.Debugw("LoadMetricsFromFile", "data", string(data))

	return err
}

// LoadMetricsFromFile Загрузка метрик из файла
func (s *Service) LoadMetricsFromFile(ctx context.Context) error {

	data, err := os.ReadFile(settings.Settings.FileStoragePath)

	if errors.Is(err, os.ErrNotExist) {
		dir, err2 := os.Getwd()
		if err2 != nil {
			return errutil.WrapError(fmt.Errorf("couldn't get the current directory, %w", err2))
		}

		logger.Log.Debug(fmt.Sprintf("file %s%s not exist", dir, settings.Settings.FileStoragePath))
		return nil

	}

	if err != nil {
		return err
	}

	return s.LoadMetricsFromData(ctx, data)

}

// Ping Проверка успешности подключения репозитория
func (s *Service) Ping(ctx context.Context) ([]byte, error) {

	return s.Repository.Ping(ctx)

}

var (
	errMetricsTypeNotFound       = errors.New("metrics type not found")
	errRetryStrategyIsNotDefined = errors.New("retry strategy is not defined")
)

func isConnectionError(err error) bool {

	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	errors.As(err, &pgErr)

	if pgErr == nil {
		return false
	}

	return pgerrcode.IsConnectionException(pgErr.Code)

}
