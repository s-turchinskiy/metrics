// Package httpresty Отправка метрики через resty
package httpresty

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/s-turchinskiy/metrics/internal/common/errutil"
	"github.com/s-turchinskiy/metrics/internal/common/hashutil"
	"github.com/s-turchinskiy/metrics/internal/common/rsautil"

	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric"
)

type ReportMetricsHTTPResty struct {
	client       *resty.Client
	url          string
	hashFunc     hashutil.HashFunc
	hashKey      string
	rsaPublicKey *rsa.PublicKey
}

func New(url string, hashFunc hashutil.HashFunc, hashKey string, rsaPublicKey *rsa.PublicKey) *ReportMetricsHTTPResty {

	return &ReportMetricsHTTPResty{
		client:       resty.New(),
		url:          url,
		hashFunc:     hashFunc,
		hashKey:      hashKey,
		rsaPublicKey: rsaPublicKey,
	}
}

func (r *ReportMetricsHTTPResty) Send(metric models.Metrics) error {

	body, err := json.Marshal(metric)
	if err != nil {
		return errutil.WrapError(fmt.Errorf("error json marshal data"))
	}

	if r.rsaPublicKey != nil {
		body, err = rsautil.Encrypt(r.rsaPublicKey, body)
		if err != nil {
			return err
		}
	}

	request := r.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body)

	if r.hashKey != "" && r.hashFunc != nil {

		hash := r.hashFunc(r.hashKey, body)
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
