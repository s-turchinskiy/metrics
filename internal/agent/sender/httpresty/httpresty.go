// Package httpresty Отправка метрики через resty
package httpresty

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/agent/sender"
	"github.com/s-turchinskiy/metrics/internal/utils/errutil"
	"github.com/s-turchinskiy/metrics/internal/utils/hashutil"
	"github.com/s-turchinskiy/metrics/internal/utils/rsautil"
)

type ReportMetricsHTTPResty struct {
	client       *resty.Client
	url          string
	hashFunc     hashutil.HashFunc
	hashKey      string
	rsaPublicKey *rsa.PublicKey
	realIP       string
}

type OptionHTTPResty func(*ReportMetricsHTTPResty)

func New(url string, opts ...OptionHTTPResty) *ReportMetricsHTTPResty {

	r := &ReportMetricsHTTPResty{
		client: resty.New(),
		url:    url,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r

}

func (r *ReportMetricsHTTPResty) Close(ctx context.Context) error {
	return nil
}

func WithHash(hashKey string, hashFunc hashutil.HashFunc) OptionHTTPResty {
	return func(r *ReportMetricsHTTPResty) {
		r.hashKey = hashKey
		r.hashFunc = hashFunc
	}
}

func WithRsaPublicKey(rsaPublicKey *rsa.PublicKey) OptionHTTPResty {
	return func(r *ReportMetricsHTTPResty) {
		r.rsaPublicKey = rsaPublicKey
	}
}

func WithRealIP(ip string) OptionHTTPResty {
	return func(r *ReportMetricsHTTPResty) {
		r.realIP = ip
	}
}

func (r *ReportMetricsHTTPResty) Send(ctx context.Context, metric models.Metrics) error {

	return r.sendAnyData(ctx, metric)

}

func (r *ReportMetricsHTTPResty) SendBatch(ctx context.Context, metrics []models.Metrics) error {
	return r.sendAnyData(ctx, metrics)
}

func (r *ReportMetricsHTTPResty) HandlerErrors(ctx context.Context, err error, data any, url string) {
	sender.HTTPHandlerErrors(err, data, url)
}

func (r *ReportMetricsHTTPResty) sendAnyData(ctx context.Context, data any) error {

	body, err := json.Marshal(data)
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

	if r.realIP != "" {
		request.SetHeader("X-Real-IP", r.realIP)
	}

	if r.hashKey != "" && r.hashFunc != nil {

		hash := r.hashFunc(r.hashKey, body)
		request.SetHeader("HashSHA256", hash)
	}

	resp, err := request.Post(r.url)

	if err != nil {
		r.HandlerErrors(ctx, err, data, r.url)
		return err
	}

	if err := sender.CheckResponseStatus(
		resp.StatusCode(),
		resp.Body(),
		r.url,
	); err != nil {
		return nil
	}

	return nil

}
