package main

import (
	"github.com/go-chi/chi/v5"
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
		panic(err)
	}
}

func run(addr *NetAddress) error {

	metricsHandler := &MetricsHandler{
		storage: &MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		},
	}

	/*router := http.NewServeMux()
	router.HandleFunc(`/update/{MetricsType}/{MetricsName}/{MetricsValue}`, metricsHandler.UpdateMetric)
	router.HandleFunc(`/value/{MetricsType}/{MetricsName}`, metricsHandler.GetMetric)
	router.HandleFunc(`/`, metricsHandler.GetAllMetrics)*/

	router := chi.NewRouter()
	router.Route("/update", func(r chi.Router) {
		r.Post("/{MetricsType}/{MetricsName}/{MetricsValue}", metricsHandler.UpdateMetric)
	})
	router.Route("/value", func(r chi.Router) {
		r.Get("/{MetricsType}/{MetricsName}", metricsHandler.GetMetric)
	})
	router.Get(`/`, metricsHandler.GetAllMetrics)

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
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	metrics, err := h.storage.GetAllMetrics()
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	io.WriteString(w, strings.Join(metrics, "\n"))

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

}

func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/plain")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
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
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(value))

}

type Metric struct {
	MetricsType  string
	MetricsName  string
	MetricsValue string
}
