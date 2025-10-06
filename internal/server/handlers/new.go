package handlers

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/repository/memcashed"
	"github.com/s-turchinskiy/metrics/internal/server/repository/postgresql"
	"github.com/s-turchinskiy/metrics/internal/server/service"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"log"
	"time"
)

type MetricsHandler struct {
	Service service.MetricsUpdater
}

const ContentTypeTextHTML = "text/html; charset=utf-8"

var (
	templateOutputAllMetrics = `<div>{{.Header}}</div><table style="margin-left: 40px">{{range $k, $v:= .Table}}<tr><td>{{$k}}</td><td>{{$v}}</td></tr>{{end}}</table>`
)

func NewHandler(ctx context.Context) *MetricsHandler {
	metricsHandler := &MetricsHandler{}
	if settings.Settings.Store == settings.Database {

		p, err := postgresql.Initialize(ctx)
		if err != nil {
			logger.Log.Debugw("Connect to database error", "error", err.Error())
			log.Fatal(err)
		}

		retryStrategy := []time.Duration{
			0,
			2 * time.Second,
			5 * time.Second}

		metricsHandler.Service = service.New(p, retryStrategy)

	} else {

		rep := &memcashed.MemCashed{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		}
		metricsHandler.Service = service.New(rep, []time.Duration{0})

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
