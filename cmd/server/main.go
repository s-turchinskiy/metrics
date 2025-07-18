package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"go.uber.org/zap"
	"log"
	"net/http"
	"strings"
	"time"
)

type MetricsHandler struct {
	storage MetricsUpdater
}

func init() {

	if err := logger.Initialize(); err != nil {
		log.Fatal(err)
	}

}

func main() {

	if err := getSettings(); err != nil {
		logger.Log.Errorw("Get Settings error", "error", err.Error())
		log.Fatal(err)
	}

	metricsHandler := &MetricsHandler{
		storage: &MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		},
	}

	if settings.Restore {
		err := metricsHandler.storage.LoadMetricsFromFile()
		if err != nil {
			logger.Log.Errorw("LoadMetricsFromFile error", "error", err.Error())
			log.Fatal(err)
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

	go saveMetricsToFilePeriodically(metricsHandler, errors)

	err := <-errors
	metricsHandler.storage.SaveMetricsToFile()
	logger.Log.Infow("error, server stopped", "error", err.Error())
	log.Fatal(err)
}

func saveMetricsToFilePeriodically(h *MetricsHandler, errors chan error) {

	if !settings.asynchronousWritingDataToFile {
		return
	}

	ticker := time.NewTicker(time.Duration(settings.StoreInterval) * time.Second)
	for range ticker.C {

		err := h.storage.SaveMetricsToFile()
		if err != nil {
			logger.Log.Infoln("error", err.Error())
			errors <- err
			return
		}
	}
}

func run(h *MetricsHandler) error {

	router := chi.NewRouter()
	router.Use(gzipMiddleware)
	router.Use(logger.Logger)
	router.Route("/update", func(r chi.Router) {
		r.Post("/", h.UpdateMetricJSON)
		r.Post("/{MetricsType}/{MetricsName}/{MetricsValue}", h.UpdateMetric)
	})
	router.Route("/value", func(r chi.Router) {
		r.Post("/", h.GetTypedMetric)
		r.Get("/{MetricsType}/{MetricsName}", h.GetMetric)
	})

	router.Get(`/`, h.GetAllMetrics)

	logger.Log.Info("Running server", zap.String("address", settings.Address.String()))

	return http.ListenAndServe(settings.Address.String(), router)
}

func gzipMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

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

		next.ServeHTTP(w, r)

	})
}
