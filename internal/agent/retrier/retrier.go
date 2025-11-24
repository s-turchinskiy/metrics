// Package retrier Задается количество попыток и задержка между попытками для отправки на сервер
package retrier

import (
	"context"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"strings"
	"time"

	"github.com/s-turchinskiy/metrics/internal/agent/logger"
)

type ReportMetricRetrier interface {
	SendWithRetries(ctx context.Context, data models.Metrics, send func(ctx context.Context, data models.Metrics) error) error
}

type ReportMetricRetry1 struct {
	ReportMetricRetrier
}

var retryIntervals = []time.Duration{
	0,
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

func (r ReportMetricRetry1) SendWithRetries(ctx context.Context, data models.Metrics, f func(ctx context.Context, metrics models.Metrics) error) error {

	var err error
	for i, delay := range retryIntervals {
		time.Sleep(delay)
		err = f(ctx, data)

		if !itIsErrorConnectionRefused(err) {
			return err
		}

		logger.Log.Infow(fmt.Sprintf("reportMetric attempt %d, server is not responding", i+1), "data", data)

	}

	return err

}

func itIsErrorConnectionRefused(err error) bool {

	return err != nil &&
		(strings.Contains(err.Error(), "connect: connection refused") || strings.Contains(err.Error(), "connection reset by peer"))
}
