package repositories

import (
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestMetricsStorage_GetMetrics(t *testing.T) {

	var val = 1.23
	var delta int64 = 2

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	tests := []struct {
		name    string
		fields  fields
		want    []models.Metrics
		wantErr bool
	}{
		{
			name: "Успешно",
			fields: fields{
				Gauge:   map[string]float64{"some": 1.23},
				Counter: map[string]int64{"some": 2},
			},
			want: []models.Metrics{
				{MType: "gauge", ID: "some", Value: &val},
				{MType: "counter", ID: "some", Delta: &delta},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetricsStorage{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			got, err := s.GetMetrics()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMetrics() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetricsStorage_UpdateMetrics(t *testing.T) {

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		metrics map[string]float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    fields
		wantErr bool
	}{
		{
			name:    "success",
			fields:  fields{Gauge: make(map[string]float64), Counter: make(map[string]int64)},
			args:    args{metrics: map[string]float64{"some": 1.23}},
			want:    fields{Gauge: map[string]float64{"some": 1.23}, Counter: map[string]int64{"PollCount": 1}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetricsStorage{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}

			err := s.UpdateMetrics(tt.args.metrics)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.want.Gauge, s.Gauge)
			assert.Equal(t, tt.want.Counter, s.Counter)
		})
	}
}
