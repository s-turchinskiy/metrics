package models

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

type StorageMetrics struct {
	Name  string
	MType string
	Delta *int64
	Value *float64
}

type UntypedMetric struct {
	MetricsType  string
	MetricsName  string
	MetricsValue string
}
