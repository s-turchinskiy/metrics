// Package reporter Отправка метрик на сервер
package reporter

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/agent/repositories"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric"
	"time"

	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/agent/retrier"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetrics"
)

type Reporter interface {
	ReportMetrics(ctx context.Context) error
	ReportMetricsBatch(cfg context.Context) error
}

type Report struct {
	storage        repositories.MetricsRepositorier
	sender         sendmetric.MetricSender
	reportInterval int
	rateLimit      int
}

func New(storage *repositories.MetricsStorage, sender sendmetric.MetricSender, reportInterval, rateLimit int) *Report {
	return &Report{
		storage:        storage,
		sender:         sender,
		reportInterval: reportInterval,
		rateLimit:      rateLimit,
	}
}

func (r *Report) ReportMetrics(ctx context.Context) error {

	ticker := time.NewTicker(time.Duration(r.reportInterval) * time.Second)
	for range ticker.C {

		select {
		case <-ctx.Done():
			return nil
		default:

			metrics, err := r.storage.GetMetrics()
			if err != nil {
				logger.Log.Infoln("failed to report metrics", err.Error())
				return err
			}

			jobs := generator(ctx, metrics)

			sendMetrics := sendmetrics.New(
				jobs,
				r.sender,
				retrier.ReportMetricRetry1{},
			)

			for w := 1; w <= r.rateLimit; w++ {
				go sendMetrics.WorkerSender(ctx)
			}

			sendMetrics.ResultHandling(ctx)
		}
	}
	return nil
}

func generator(ctx context.Context, input []models.Metrics) chan models.Metrics {
	inputCh := make(chan models.Metrics, len(input))

	go func() {
		defer close(inputCh)

		for _, data := range input {
			select {
			case <-ctx.Done():
				return
			case inputCh <- data:
			}
		}
	}()

	return inputCh
}
