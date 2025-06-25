package main

import (
	"net/http"
	"strings"
)

func main() {
	metricsHandler := &MetricsHandler{
		storage: &MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
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
		w.Header().Set("Content-Type", "text/plain")
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
		MetricsValue: pathSlice[2], //r.PathValue("MetricsValue"),
	}

	err := h.storage.UpdateMetric(metric)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "text/plain")
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
