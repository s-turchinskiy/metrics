// Package handlers Обработка входящих http-запросов
package handlers

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/server/repository"
	"log"
	"time"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/service"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
)

type MetricsHandler struct {
	Service                       service.MetricsUpdater
	asynchronousWritingDataToFile bool
}

const (
	ContentTypeTextHTML         = "text/html; charset=utf-8"
	ContentTypeTextPlain        = "text/plain"
	ContentTypeTextPlainCharset = "text/plain; charset=utf-8"
	ContentTypeApplicationJSON  = "application/json"

	TextErrorGettingData = "error getting data"
)

var (
	templateOutputAllMetrics = `<div>{{.Header}}</div><table style="margin-left: 40px">{{range $k, $v:= .Table}}<tr><td>{{$k}}</td><td>{{$v}}</td></tr>{{end}}</table>`
)

func NewHandler(
	ctx context.Context,
	rep repository.Repository,
	fileStoragePath string,
	asynchronousWritingDataToFile bool) *MetricsHandler {
	metricsHandler := &MetricsHandler{asynchronousWritingDataToFile: asynchronousWritingDataToFile}
	if settings.Settings.Store == settings.Database {

		retryStrategy := []time.Duration{
			0,
			2 * time.Second,
			5 * time.Second}

		metricsHandler.Service = service.New(rep, retryStrategy, fileStoragePath)

	} else {

		metricsHandler.Service = service.New(rep, []time.Duration{0}, fileStoragePath)

		if settings.Settings.Restore {
			err := metricsHandler.Service.LoadMetricsFromFile(ctx)
			if err != nil {
				logger.Log.Errorw("LoadMetricsFromFile error", "error", err.Error())
				log.Fatal(err)
			}
		}
	}

	return metricsHandler

}
