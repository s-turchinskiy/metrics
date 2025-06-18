package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func main() {
	metricsHandler := &MetricsHandler{
		storage: &MetricsStorage{
			metrics: make(map[string]map[string]any),
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc(`/update/{MetricsType}/{MetricsName}/{MetricsValue}`, metricsHandler.UpdateMetric)
	mux.HandleFunc(`/get/{MetricsType}/{MetricsName}`, metricsHandler.GetMetric)
	mux.HandleFunc(`/`, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}

type MetricsUpdater interface {
	UpdateMetric(metric Metric) error
	GetMetric(metric Metric) (string, error)
}

type MetricsHandler struct {
	storage MetricsUpdater
}

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/update/")
	pathSlice := strings.Split(path, "/")

	metric := Metric{
		MetricsType:  pathSlice[0],
		MetricsName:  pathSlice[1],
		MetricsValue: r.PathValue("MetricsValue"),
	}

	err := h.storage.UpdateMetric(metric)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)

}

func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/get/")
	pathSlice := strings.Split(path, "/")

	metric := Metric{
		MetricsType: pathSlice[0],
		MetricsName: pathSlice[1],
	}

	value, err := h.storage.GetMetric(metric)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(value))

}

type Metric struct {
	MetricsType  string
	MetricsName  string
	MetricsValue string
}

type MetricsStorage struct {
	metrics map[string]map[string]any
}

func (s *MetricsStorage) UpdateMetric(metric Metric) error {

	switch metricsType := metric.MetricsType; metricsType {
	case "gauge":

		value, err := strconv.ParseFloat(metric.MetricsValue, 64)
		if err != nil {
			return err
		}

		if s.metrics[metric.MetricsType] == nil {
			s.metrics[metric.MetricsType] = make(map[string]any)
		}
		s.metrics[metric.MetricsType][metric.MetricsName] = value
	case "counter":

		value, err := strconv.ParseInt(metric.MetricsValue, 10, 64)
		if err != nil {
			return err
		}

		if s.metrics[metric.MetricsType] == nil {
			s.metrics[metric.MetricsType] = make(map[string]any)
		}

		if s.metrics[metric.MetricsType][metric.MetricsName] == nil {
			s.metrics[metric.MetricsType][metric.MetricsName] = value
		} else {
			s.metrics[metric.MetricsType][metric.MetricsName] =
				s.metrics[metric.MetricsType][metric.MetricsName].(int64) + value
		}

	default:
		return errMetricsTypeNotFound
	}

	return nil
}

func (s *MetricsStorage) GetMetric(metric Metric) (string, error) {

	if s.metrics[metric.MetricsType] == nil {
		s.metrics[metric.MetricsType] = make(map[string]any)
	}

	value := s.metrics[metric.MetricsType][metric.MetricsName]

	if value == nil {
		return "0", nil
	}

	switch metricsType := metric.MetricsType; metricsType {
	case "gauge":

		convValue, ok := value.(float64)
		if !ok {
			return "", errCantConvertToString
		}

		return fmt.Sprintf("%f", convValue), nil
	case "counter":

		convValue, ok := value.(int64)
		if !ok {
			return "", errCantConvertToString
		}

		return strconv.FormatInt(convValue, 10), nil

	default:
		return "", errMetricsTypeNotFound
	}
}

var (
	errMetricsTypeNotFound = errors.New("metrics type not found")
	errCantConvertToString = errors.New("can't convert to a string")
)
