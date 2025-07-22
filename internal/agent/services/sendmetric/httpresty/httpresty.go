package httpresty

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"net/http"
)

type ReportMetricsHttpResty struct {
	client        *resty.Client
	serverAddress string
}

func New(serverAddress string) *ReportMetricsHttpResty {

	return &ReportMetricsHttpResty{
		client:        resty.New(),
		serverAddress: serverAddress,
	}
}

func (r *ReportMetricsHttpResty) Send(metric models.Metrics) error {

	url := fmt.Sprintf("%s/update/", r.serverAddress)
	resp, err := r.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metric).
		Post(url)

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
		return err
	}

	if resp.StatusCode() != http.StatusOK {

		logger.Log.Infow("error. status code <> 200",
			"status code", resp.StatusCode(),
			"url", url,
			"body", string(resp.Body()))
		err := fmt.Errorf("status code <> 200, = %d, url : %s", resp.StatusCode(), url)
		return err
	}

	return nil

}
