package sendmetrics

import (
	"errors"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/agent/retrier"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric"
	"sync"
)

type MetricsSender interface {
	Send() []error
	ErrorHandling([]error)
}

type SendMetrics struct {
	MetricsSender
	sender  sendmetric.MetricSender
	retrier retrier.ReportMetricRetrier
	metrics []models.Metrics
}

func New(metrics []models.Metrics, sender sendmetric.MetricSender, retrier retrier.ReportMetricRetrier) *SendMetrics {
	return &SendMetrics{
		metrics: metrics,
		sender:  sender,
		retrier: retrier,
	}
}

func (s *SendMetrics) ErrorHandling(errs []error) {

	if len(errs) != 0 {
		logger.Log.Info("Unsuccess ReportMetrics")
		logger.Log.Info("errors sending data ", errors.Join(errs...))
	} else {
		logger.Log.Info("Success ReportMetrics")
	}

}
func (s *SendMetrics) Send() []error {

	result := make(chan error, len(s.metrics))
	wg := sync.WaitGroup{}
	wg.Add(len(s.metrics))

	for _, metric := range s.metrics {
		go func() {
			defer wg.Done()
			if s.retrier != nil {
				result <- s.retrier.SendWithRetries(metric, s.sender.Send)
			} else {
				result <- s.sender.Send(metric)
			}
		}()
	}

	wg.Wait()
	close(result)

	var errs []error
	for err := range result {
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs

}
