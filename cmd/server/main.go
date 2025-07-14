package main

import (
	"database/sql"
	"fmt"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"sync"
	"time"
)

type MetricsHandler struct {
	storage MetricsUpdater
	db      *sql.DB
}

func init() {

	if err := logger.Initialize(); err != nil {
		panic(err)
	}

}

func main() {

	if err := getSettings(); err != nil {
		logger.Log.Errorw("Get Settings error", "error", err.Error())
		panic(err)
	}

	db, err := connectToStore()
	if err != nil {
		logger.Log.Debugw("Connect to database error", "error", err.Error())
		//logger.Log.Errorw("Connect to database error", "error", err.Error())
		//panic(err)
	}

	metricsHandler := &MetricsHandler{
		storage: &MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
			mutex:   sync.Mutex{},
		},
		db: db,
	}

	defer metricsHandler.db.Close()

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

	go saveMetricsToFilePeriodically(metricsHandler, errors)

	err = <-errors
	metricsHandler.storage.SaveMetricsToFile()
	logger.Log.Infow("error, server stopped", "error", err.Error())
	panic(err)
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
	router.Route("/update", func(r chi.Router) {
		r.Post("/", logger.WithLogging(gzipMiddleware(h.UpdateMetricJSON)))
		r.Post("/{MetricsType}/{MetricsName}/{MetricsValue}", logger.WithLogging(gzipMiddleware(h.UpdateMetric)))
	})
	router.Route("/value", func(r chi.Router) {
		r.Post("/", logger.WithLogging(gzipMiddleware(h.GetTypedMetric)))
		r.Get("/{MetricsType}/{MetricsName}", logger.WithLogging(gzipMiddleware(h.GetMetric)))
	})
	router.Route("/ping", func(r chi.Router) {
		r.Get("/", logger.WithLogging(gzipMiddleware(h.Ping)))
	})
	router.Get(`/`, logger.WithLogging(gzipMiddleware(h.GetAllMetrics)))

	logger.Log.Info("Running server", zap.String("address", settings.Address.String()))

	return http.ListenAndServe(settings.Address.String(), router)
}

func connectToStore() (*sql.DB, error) {
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		settings.Database.Host, settings.Database.Login, settings.Database.Password, settings.Database.DBName)

	db, err := sql.Open("pgx", ps)
	if err != nil {
		return nil, err
	}

	/*err = db.Ping()
	if err != nil {
		return nil, err
	}*/

	return db, nil

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
