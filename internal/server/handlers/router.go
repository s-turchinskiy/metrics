package handlers

import (
	"github.com/go-chi/chi/v5"
	_ "github.com/s-turchinskiy/metrics/internal/server/handlers/swagger"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/gzip"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/hash"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @Title MetricStorage API
// @Description Сервис хранения метрик.
// @Version 1.0

// @Contact.email s.turchinskiy@yandex.ru

// @BasePath /
// @Host nohost.io:8080

// @SecurityDefinitions.apikey ApiKeyAuth
// @In header
// @Name authorization

// @Tag.name Info
// @Tag.description "Группа запросов метрик"

// @Tag.name Update
// @Tag.description "Группа обновления метрик"

// @Tag.name Ping
// @Tag.description "Группа проверки работоспособности сервиса"

func Router(h *MetricsHandler) chi.Router {

	router := chi.NewRouter()
	router.Use(logger.Logger)
	router.Use(hash.HashWriteMiddleware)
	router.Use(hash.HashReadMiddleware)
	router.Use(gzip.GzipMiddleware)
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
	router.Mount("/swagger", httpSwagger.WrapHandler)

	return router

}
