// Package sender Интерфейс отправки метрики и 2 общих метода
package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"net/http"

	"github.com/s-turchinskiy/metrics/internal/agent/logger"
)

type MetricSender interface {
	Send(context.Context, models.Metrics) error
	SendBatch(context.Context, []models.Metrics) error
	HandlerErrors(ctx context.Context, err error, data any, url string)
	Close(ctx context.Context) error
}

func HTTPHandlerErrors(err error, data any, url string) {

	if err != nil {

		text := err.Error()
		var bytes []byte
		bytes, err2 := json.Marshal(data)
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
