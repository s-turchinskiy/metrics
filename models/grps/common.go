package proto

import "github.com/s-turchinskiy/metrics/internal/server/models"

func (m *Metric) SetMTypeFromString(s string) *Metric {
	switch s {
	case "gauge":
		m.MType = Metric_MetricType(0)
	case "counter":
		m.MType = Metric_MetricType(1)
	}

	return m
}

func (m *Metric) GetMTypeAsString() string {
	switch m.MType {
	case 0:
		return "gauge"
	case 1:
		return "counter"
	}

	return ""
}

func (m *Metric) GetStorageMetric() models.StorageMetrics {

	return models.StorageMetrics{
		MType: m.GetMTypeAsString(),
		Name:  m.Id,
		Value: &m.Value,
		Delta: &m.Delta,
	}
}
