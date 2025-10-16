// Package httpresty Отправка метрики через resty
package httpresty

import (
	"encoding/json"
	"fmt"
	error2 "github.com/s-turchinskiy/metrics/internal/common/error"
	"github.com/s-turchinskiy/metrics/internal/common/hash"

	"github.com/go-resty/resty/v2"

	"github.com/s-turchinskiy/metrics/cmd/agent/config"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric"
)

type ReportMetricsHTTPResty struct {
	client   *resty.Client
	url      string
	hashFunc hash.HashFunc
}

func New(url string, hashFunc hash.HashFunc) *ReportMetricsHTTPResty {

	return &ReportMetricsHTTPResty{
		client:   resty.New(),
		url:      url,
		hashFunc: hashFunc,
	}
}

func (r *ReportMetricsHTTPResty) Send(metric models.Metrics) error {

	body, err := json.Marshal(metric)
	if err != nil {
		return error2.WrapError(fmt.Errorf("error json marshal data"))
	}

	request := r.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body)

	if config.HashKey != "" && r.hashFunc != nil {

		hash := r.hashFunc(config.HashKey, body)
		request.SetHeader("HashSHA256", hash)
	}

	resp, err := request.Post(r.url)

	if err != nil {
		sendmetric.HandlerErrors(err, metric, r.url)
		return err
	}

	if err := sendmetric.CheckResponseStatus(
		resp.StatusCode(),
		resp.Body(),
		r.url,
	); err != nil {
		return nil
	}

	return nil

}
