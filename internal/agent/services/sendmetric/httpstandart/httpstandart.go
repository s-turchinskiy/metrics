package httpstandart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/s-turchinskiy/metrics/cmd/agent/config"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric"
	"github.com/s-turchinskiy/metrics/internal/common"
)

type ReportMetricsHTTPStandart struct {
	url      string
	hashFunc common.HashFunc
}

func New(url string, hashFunc common.HashFunc) *ReportMetricsHTTPStandart {

	return &ReportMetricsHTTPStandart{
		url:      url,
		hashFunc: hashFunc,
	}
}

func (r *ReportMetricsHTTPStandart) Send(metric models.Metrics) error {

	data, err := json.Marshal(metric)
	if err != nil {
		return common.WrapError(fmt.Errorf("error json marshal data"))
	}

	client := new(http.Client)
	request, _ := http.NewRequest("POST", r.url, bytes.NewReader(data))
	request.Header.Add("Content-Type", "application/json")

	if config.HashKey != "" && r.hashFunc != nil {

		hash := r.hashFunc(config.HashKey, data)
		request.Header.Add("HashSHA256", hash)
	}

	resp, err := client.Do(request)

	if err != nil {
		sendmetric.HandlerErrors(err, metric, r.url)
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Debugw(common.WrapError(fmt.Errorf("error read body")).Error())
		return err
	}

	resp.Body.Close()

	if err := sendmetric.CheckResponseStatus(
		resp.StatusCode,
		body,
		r.url,
	); err != nil {
		return nil
	}

	return nil

}
