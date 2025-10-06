package reporter

import (
	"fmt"
	"github.com/s-turchinskiy/metrics/cmd/agent/config"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/agent/retrier"
	"github.com/s-turchinskiy/metrics/internal/agent/services"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric/httpresty"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetrics"
	"github.com/s-turchinskiy/metrics/internal/common"
	"time"
)

func main() {

}

func ReportMetrics(h *services.MetricsHandler, errorsChan chan error, doneCh chan struct{}) {

	ticker := time.NewTicker(time.Duration(config.ReportInterval) * time.Second)
	for range ticker.C {

		metrics, err := h.Storage.GetMetrics()
		if err != nil {
			logger.Log.Infoln("failed to report metrics", err.Error())
			errorsChan <- err
			return
		}

		jobs := generator(doneCh, metrics)

		sender := httpresty.New(
			fmt.Sprintf("%s/update/", h.ServerAddress),
			common.СomputeHexadecimalSha256Hash,
		)

		sendMetrics := sendmetrics.New(
			jobs,
			doneCh,
			sender,
			retrier.ReportMetricRetry1{},
		)

		for w := 1; w <= config.RateLimit; w++ {
			go sendMetrics.WorkerSender()
		}

		sendMetrics.ResultHandling()

	}
}

func generator(doneCh chan struct{}, input []models.Metrics) chan models.Metrics {
	inputCh := make(chan models.Metrics, len(input))

	go func() {
		defer close(inputCh)

		for _, data := range input {
			select {
			case <-doneCh:
				return
			case inputCh <- data:
			}
		}
	}()

	return inputCh
}
