package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/mailru/easyjson"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

func init() {

	if err := logger.Initialize(); err != nil {
		panic(err)
	}

	if err := getSettings(); err != nil {
		logger.Log.Errorw("Get Settings error", "error", err.Error())
		panic(err)
	}

}

func main() {

	metricsHandler := &MetricsHandler{
		storage: &MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
			mutex:   sync.Mutex{},
		},
	}

	if settings.Restore {
		err := metricsHandler.storage.LoadMetricsFromFile()
		if err != nil {
			logger.Log.Errorw("LoadMetricsFromFile error", "error", err.Error())
			panic(err)
		}
	}

	errors := make(chan error)

	go func() {
		err := run(metricsHandler)
		if err != nil {

			logger.Log.Errorw("Server startup error", "error", err.Error())
			errors <- err
			return
		}
	}()

	if settings.asynchronousWritingDataToFile {

		go func() {

			for {

				time.Sleep(time.Duration(settings.StoreInterval) * time.Second)

				err := metricsHandler.storage.SaveMetricsToFile()
				if err != nil {
					logger.Log.Infoln("error", err.Error())
					errors <- err
					return
				}

			}

		}()
	}

	err := <-errors
	metricsHandler.storage.SaveMetricsToFile()
	logger.Log.Infow("error, server stopped", "error", err.Error())
	panic(err)
}

func run(metricsHandler *MetricsHandler) error {

	router := chi.NewRouter()
	router.Route("/update", func(r chi.Router) {
		r.Post("/", logger.WithLogging(gzipMiddleware(metricsHandler.UpdateMetricJSON)))
		r.Post("/{MetricsType}/{MetricsName}/{MetricsValue}", logger.WithLogging(gzipMiddleware(metricsHandler.UpdateMetric)))
	})
	router.Route("/value", func(r chi.Router) {
		r.Post("/", logger.WithLogging(gzipMiddleware(metricsHandler.GetTypedMetric)))
		r.Get("/{MetricsType}/{MetricsName}", logger.WithLogging(gzipMiddleware(metricsHandler.GetMetric)))
	})
	router.Get(`/`, logger.WithLogging(gzipMiddleware(metricsHandler.GetAllMetrics)))

	logger.Log.Info("Running server", zap.String("address", settings.Address.String()))

	return http.ListenAndServe(settings.Address.String(), router)
}

type MetricsHandler struct {
	storage MetricsUpdater
}

func (h *MetricsHandler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	if r.URL.Path != "/" {
		logger.Log.Infow("error, Path != \"/\"", "path", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		logger.Log.Infow("error, Method != Get", "Method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	result := h.storage.GetAllMetrics()

	for mtype, table := range result {
		io.WriteString(w, fmt.Sprintf("<div>%s</div>", mtype))
		io.WriteString(w, "<table style=\"margin-left: 40px\">")
		for name, value := range table {
			io.WriteString(w, fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", name, value))
		}
		io.WriteString(w, "</table>")
	}

	/*for _, str := range metrics {
		runes := []rune(str)
		if runes[0] == '\t' {
			io.WriteString(w, fmt.Sprintf("%s", str))
		} else {

		}
	}*/

}

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		logger.Log.Infow("error, Method != Post", "Method", r.Method)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/update/")
	pathSlice := strings.Split(path, "/")

	metric := models.UntypedMetric{
		MetricsType:  pathSlice[0],
		MetricsName:  pathSlice[1],
		MetricsValue: pathSlice[2], //r.PathValue("MetricsValue"),
	}

	err := h.storage.UpdateMetric(metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

}

func (h *MetricsHandler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		logger.Log.Infow("error, Method != Post", "Method", r.Method)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Info("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	metric := models.StorageMetrics{Name: req.ID, MType: req.MType, Delta: req.Delta, Value: req.Value}
	result, err := h.storage.UpdateTypedMetric(metric)
	if err != nil {
		logger.Log.Infoln("error", err.Error())
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := models.Metrics{ID: result.Name, MType: result.MType, Delta: result.Delta, Value: result.Value}
	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		logger.Log.Info("error encoding response", zap.Error(err))
		return
	}

	if !settings.asynchronousWritingDataToFile {
		err := h.storage.SaveMetricsToFile()
		if err != nil {
			logger.Log.Info("error SaveMetricsToFile", zap.Error(err))
			return

		}
	}

}

func (h *MetricsHandler) GetTypedMetric(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		logger.Log.Infow("error, Method != Post", "Method", r.Method)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Info("cannot decode request JSON body", zap.Error(err))
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	metric := models.StorageMetrics{Name: req.ID, MType: req.MType, Delta: req.Delta, Value: req.Value}

	result, err := h.storage.GetTypedMetric(metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := &models.Metrics{ID: result.Name, MType: result.MType, Delta: result.Delta, Value: result.Value}
	rawBytes, err := easyjson.Marshal(resp)
	if err != nil {
		logger.Log.Info("error encoding response", zap.Error(err))
		return
	}
	w.Write(rawBytes)

}

func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html")

	if r.Method != http.MethodGet {
		logger.Log.Infow("error, Method != Get", "Method", r.Method)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/value/")
	pathSlice := strings.Split(path, "/")

	metric := models.UntypedMetric{
		MetricsType: pathSlice[0],
		MetricsName: pathSlice[1],
	}

	value, err := h.storage.GetMetric(metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(value))

}

func gzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")

		if supportsGzip {
			cw := newCompressWriter(w)
			w = cw
			defer cw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(w, r)
	}
}
