// Package models Модели
package models

//go:generate easyjson models.go

// Metrics содержит запрос и ответ на обновление данных метрики.
//
//easyjson:json
type Metrics struct {
	ID    string   `json:"id" enums:"counter,gauge" example:"gauge"` // имя метрики
	MType string   `json:"type" example:"Alloc"`                     // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty" example:"100"`            // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty" example:"6649272"`        // значение метрики в случае передачи gauge
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

type DatabaseTableGauges struct {
	MetricsName string  `db:"metrics_name"`
	Value       float64 `db:"value"`
}
