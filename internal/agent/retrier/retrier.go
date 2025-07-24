package retrier

import (
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"strings"
	"time"
)

type ReportMetricRetrier interface {
	SendWithRetries(models.Metrics, func(models.Metrics) error) error
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

func (r ReportMetricRetry1) SendWithRetries(metric models.Metrics, f func(models.Metrics) error) error {

	var err error
	for i, delay := range retryIntervals {
		time.Sleep(delay)
		err = f(metric)

		if !itIsErrorConnectionRefused(err) {
			return err
		}

		logger.Log.Infow(fmt.Sprintf("reportMetric attempt %d, server is not responding", i+1), "data", metric)

	}

	return err

}

func itIsErrorConnectionRefused(err error) bool {

	return err != nil &&
		(strings.Contains(err.Error(), "connect: connection refused") || strings.Contains(err.Error(), "connection reset by peer"))
}
