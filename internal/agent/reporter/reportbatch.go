package reporter

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/s-turchinskiy/metrics/cmd/agent/config"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/services"
	"net/http"
	"time"
)

func ReportMetricsBatch(h *services.MetricsHandler, errors chan error) {

	url := fmt.Sprintf("%s/updates/", h.ServerAddress)

	ticker := time.NewTicker(time.Duration(config.ReportInterval) * time.Second)
	for range ticker.C {

		client := resty.New()

		metrics, err := h.Storage.GetMetrics()
		if err != nil {
			logger.Log.Infoln("failed to report metrics batch", err.Error())
			errors <- err
			return
		}

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(metrics).
			Post(url)

		if err != nil {

			var bytes []byte
			bytes, err2 := json.Marshal(metrics)
			if err2 != nil {
				logger.Log.Infow("conversion error metric",
					"error", err2.Error(),
					"url", url,
				)
			}

			logger.Log.Infow("error sending request",
				"error", err.Error(),
				"url", url,
				"body", string(bytes))

			errors <- err
			return
		}

		if resp.StatusCode() != http.StatusOK {

			logger.Log.Infow("error. status code <> 200",
				"status code", resp.StatusCode(),
				"url", url,
				"body", string(resp.Body()))
			err := fmt.Errorf("status code <> 200, = %d, url : %s", resp.StatusCode(), url)

			errors <- err
			return

		}

		logger.Log.Info("Success ReportMetricsBatch ", string(resp.Body()))
	}
}
