// Package httpstandart Отправка метрики через стандартный http клиент
package httpstandart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/agent/sender"
	"github.com/s-turchinskiy/metrics/internal/utils/errutil"
	"github.com/s-turchinskiy/metrics/internal/utils/hashutil"
	"io"
	"net/http"

	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
)

type ReportMetricsHTTPStandart struct {
	url      string
	hashFunc hashutil.HashFunc
	hashKey  string
}

func New(url string, hashFunc hashutil.HashFunc, hashKey string) *ReportMetricsHTTPStandart {

	return &ReportMetricsHTTPStandart{
		url:      url,
		hashFunc: hashFunc,
		hashKey:  hashKey,
	}
}

func (r *ReportMetricsHTTPStandart) Send(metric models.Metrics) error {

	data, err := json.Marshal(metric)
	if err != nil {
		return errutil.WrapError(fmt.Errorf("error json marshal data"))
	}

	client := new(http.Client)
	request, _ := http.NewRequest("POST", r.url, bytes.NewReader(data))
	request.Header.Add("Content-Type", "application/json")

	if r.hashKey != "" && r.hashFunc != nil {

		hash := r.hashFunc(r.hashKey, data)
		request.Header.Add("HashSHA256", hash)
	}

	resp, err := client.Do(request)

	if err != nil {
		sender.HTTPHandlerErrors(err, metric, r.url)
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Debugw(errutil.WrapError(fmt.Errorf("error read body")).Error())
		return err
	}

	resp.Body.Close()

	if err := sender.CheckResponseStatus(
		resp.StatusCode,
		body,
		r.url,
	); err != nil {
		return nil
	}

	return nil

}
