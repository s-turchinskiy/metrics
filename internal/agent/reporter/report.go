// Package reporter Отправка метрик на сервер
package reporter

import (
	"context"
	"crypto/rsa"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/common/hashutil"
	"time"

	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/agent/retrier"
	"github.com/s-turchinskiy/metrics/internal/agent/services"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric/httpresty"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetrics"
)

func ReportMetrics(ctx context.Context,
	h *services.MetricsHandler,
	reportInterval,
	rateLimit int,
	hashKey string,
	rsaPublicKey *rsa.PublicKey,
	errorsChan chan error) {

	ticker := time.NewTicker(time.Duration(reportInterval) * time.Second)
	for range ticker.C {

		select {
		case <-ctx.Done():
			return
		default:

			metrics, err := h.Storage.GetMetrics()
			if err != nil {
				logger.Log.Infoln("failed to report metrics", err.Error())
				errorsChan <- err
				return
			}

			jobs := generator(ctx, metrics)

			sender := httpresty.New(
				fmt.Sprintf("%s/update/", h.ServerAddress),
				hashutil.СomputeHexadecimalSha256Hash,
				hashKey,
				rsaPublicKey,
			)

			sendMetrics := sendmetrics.New(
				jobs,
				sender,
				retrier.ReportMetricRetry1{},
			)

			for w := 1; w <= rateLimit; w++ {
				go sendMetrics.WorkerSender(ctx)
			}

			sendMetrics.ResultHandling(ctx)
		}
	}
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
