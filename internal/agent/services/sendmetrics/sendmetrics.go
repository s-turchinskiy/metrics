// Package sendmetrics Воркер отправки метрик
package sendmetrics

import (
	"context"
	"errors"
	"github.com/s-turchinskiy/metrics/internal/agent/sender"

	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/agent/retrier"
)

type MetricsSender interface {
	WorkerSender()
	ResultHandling()
}

type SendMetrics struct {
	MetricsSender
	numJobs int
	jobs    <-chan models.Metrics
	results chan error
	sender  sender.MetricSender
	retrier retrier.ReportMetricRetrier
}

func New(
	jobs <-chan models.Metrics,
	sender sender.MetricSender,
	retrier retrier.ReportMetricRetrier) *SendMetrics {

	return &SendMetrics{
		numJobs: cap(jobs),
		jobs:    jobs,
		results: make(chan error, cap(jobs)),
		sender:  sender,
		retrier: retrier,
	}
}

func (s *SendMetrics) ResultHandling(ctx context.Context) {

	var err error
	var errs []error
	for a := 1; a <= s.numJobs; a++ {
		select {
		case <-ctx.Done():
			return
		case err = <-s.results:
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	close(s.results)

	if len(errs) != 0 {
		logger.Log.Info("Unsuccess ReportMetrics")
		logger.Log.Info("errors sending data ", errors.Join(errs...))
	} else {
		logger.Log.Info("Success ReportMetrics")
	}

}

func (s *SendMetrics) WorkerSender(ctx context.Context) {

	for metric := range s.jobs {

		select {
		case <-ctx.Done():
			return
		default:
			if s.retrier != nil {
				s.results <- s.retrier.SendWithRetries(ctx, metric, s.sender.Send)
			} else {
				s.results <- s.sender.Send(ctx, metric)
			}
		}
	}
}
