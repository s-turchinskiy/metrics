package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/s-turchinskiy/metrics/internal/server/gzip"
	"github.com/s-turchinskiy/metrics/internal/server/handlers"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
)

func Router(h *handlers.MetricsHandler) chi.Router {

	router := chi.NewRouter()
	router.Use(gzip.GzipMiddleware)
	router.Use(logger.Logger)
	router.Route("/update", func(r chi.Router) {
		r.Post("/", h.UpdateMetricJSON)
		r.Post("/{MetricsType}/{MetricsName}/{MetricsValue}", h.UpdateMetric)
	})
	router.Route("/updates", func(r chi.Router) {
		r.Post("/", h.UpdateMetricsBatch)
	})
	router.Route("/value", func(r chi.Router) {
		r.Post("/", h.GetTypedMetric)
		r.Get("/{MetricsType}/{MetricsName}", h.GetMetric)
	})
	router.Route("/ping", func(r chi.Router) {
		r.Get("/", h.Ping)
	})

	router.Get(`/`, h.GetAllMetrics)

	return router

}
