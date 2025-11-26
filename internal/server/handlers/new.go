// Package handlers Обработка входящих http-запросов
package handlers

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/server/service"
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
	service *service.Service,
	asynchronousWritingDataToFile bool) *MetricsHandler {
	metricsHandler := &MetricsHandler{
		asynchronousWritingDataToFile: asynchronousWritingDataToFile,
		Service:                       service,
	}

	return metricsHandler

}
