package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/s-turchinskiy/metrics/cmd/server/internal/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
)

type NetAddress struct {
	Host string
	Port int
}

func main() {

	addr := NetAddress{Host: "localhost", Port: 8080}
	parseFlags(&addr)
	err := run(&addr)
	if err != nil {

		logger.Log.Errorw("Server startup error", "error", err.Error())
		panic(err)
	}
}

func run(addr *NetAddress) error {

	if err := logger.Initialize(); err != nil {
		return err
	}

	metricsHandler := &MetricsHandler{
		storage: &MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		},
	}

	router := chi.NewRouter()
	router.Route("/update", func(r chi.Router) {
		r.Post("/{MetricsType}/{MetricsName}/{MetricsValue}", logger.WithLogging(metricsHandler.UpdateMetric))
	})
	router.Route("/value", func(r chi.Router) {
		r.Get("/{MetricsType}/{MetricsName}", logger.WithLogging(metricsHandler.GetMetric))
	})
	router.Get(`/`, logger.WithLogging(metricsHandler.GetAllMetrics))

	logger.Log.Info("Running server", zap.String("address", addr.String()))

	return http.ListenAndServe(addr.String(), router)
}

type MetricsUpdater interface {
	UpdateMetric(metric Metric) error
	GetMetric(metric Metric) (string, error)
	GetAllMetrics() ([]string, error)
}

type MetricsHandler struct {
	storage MetricsUpdater
}

func (h *MetricsHandler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" {
		logger.Log.Errorw("Path != \"/\"", "path", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	metrics, err := h.storage.GetAllMetrics()
	if err != nil {
		logger.Log.Error(err.Error())
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	io.WriteString(w, strings.Join(metrics, "\n"))

}

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		logger.Log.Error(err.Error())
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
		logger.Log.Error(err.Error())
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

}

func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/plain")

	if err := r.ParseForm(); err != nil {
		logger.Log.Error(err.Error())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/value/")
	pathSlice := strings.Split(path, "/")

	metric := Metric{
		MetricsType: pathSlice[0],
		MetricsName: pathSlice[1],
	}

	value, err := h.storage.GetMetric(metric)
	if err != nil {
		logger.Log.Error(err.Error())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
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
