// Package sendmetric Интерфейс отправки метрики и 2 общих метода
package sendmetric

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
)

type MetricSender interface {
	Send(models.Metrics) error
}

func HandlerErrors(err error, metric models.Metrics, url string) {

	if err != nil {

		text := err.Error()
		var bytes []byte
		bytes, err2 := json.Marshal(metric)
		if err2 != nil {
			logger.Log.Infow("conversion error metric",
				"error", err2.Error(),
				"url", url,
			)
		}

		logger.Log.Infow("error sending request",
			"error", text,
			"url", url,
			"body", string(bytes))
	}

}

func CheckResponseStatus(statusCode int, body []byte, url string) error {

	if statusCode != http.StatusOK {

		logger.Log.Infow("error. status code <> 200",
			"status code", statusCode,
			"url", url,
			"body", string(body))
		err := fmt.Errorf("status code <> 200, = %d, url : %s", statusCode, url)
		return err
	}

	return nil
}
