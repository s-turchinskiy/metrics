package sendmetric

import "github.com/s-turchinskiy/metrics/internal/agent/models"

type MetricSender interface {
	Send(models.Metrics) error
}
