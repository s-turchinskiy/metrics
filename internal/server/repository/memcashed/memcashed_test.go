package memcashed

//lint:ignore

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestMemCashed_CountCounters(t *testing.T) {

	counterWithValue := make(map[string]int64)
	counterWithValue["count1"] = 1

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name:   "Количество Counter = 0",
			fields: fields{Counter: make(map[string]int64)},
			args:   args{ctx: context.Background()},
			want:   0,
		},
		{
			name:   "Количество Counter = 1",
			fields: fields{Counter: counterWithValue},
			args:   args{ctx: context.Background()},
			want:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			if got := m.CountCounters(tt.args.ctx); got != tt.want {
				t.Errorf("CountCounters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemCashed_CountGauges(t *testing.T) {

	gaugeWithValue := make(map[string]float64)
	gaugeWithValue["gauge1"] = 1

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name:   "Количество Gauge = 0",
			fields: fields{Gauge: make(map[string]float64)},
			args:   args{ctx: context.Background()},
			want:   0,
		},
		{
			name:   "Количество Gauge = 1",
			fields: fields{Gauge: gaugeWithValue},
			args:   args{ctx: context.Background()},
			want:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			if got := m.CountGauges(tt.args.ctx); got != tt.want {
				t.Errorf("CountGauges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemCashed_GetAllCounters(t *testing.T) {

	counterWithValue := make(map[string]int64)
	counterWithValue["name"] = 1

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]int64
		wantErr bool
	}{
		{
			name:   "Получение всех Counter",
			fields: fields{Counter: counterWithValue},
			args: args{
				ctx: context.Background(),
			},
			want:    counterWithValue,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			got, err := m.GetAllCounters(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllCounters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAllCounters() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemCashed_GetAllGauges(t *testing.T) {

	gaugeWithValue := make(map[string]float64)
	gaugeWithValue["name"] = 1

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]float64
		wantErr bool
	}{
		{
			name:   "Получение всех Gauge",
			fields: fields{Gauge: gaugeWithValue},
			args: args{
				ctx: context.Background(),
			},
			want:    gaugeWithValue,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			got, err := m.GetAllGauges(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllGauges() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAllGauges() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemCashed_GetGauge(t *testing.T) {

	gaugeWithValue := make(map[string]float64)
	gaugeWithValue["name"] = 1

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx         context.Context
		metricsName string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantValue float64
		wantExist bool
		wantErr   bool
	}{
		{
			name:      "Получение Gauge",
			fields:    fields{Gauge: gaugeWithValue},
			args:      args{ctx: context.Background(), metricsName: "name"},
			wantValue: 1,
			wantExist: true,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			value, exist, err := m.GetGauge(tt.args.ctx, tt.args.metricsName)

			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantExist, exist)
			assert.NoError(t, err)

		})
	}
}

func TestMemCashed_Ping(t *testing.T) {
	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			want:    nil,
			wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			got, err := m.Ping(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Ping() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemCashed_ReloadAllCounters(t *testing.T) {
	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx      context.Context
		newValue map[string]int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			args:    args{newValue: make(map[string]int64)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			if err := m.ReloadAllCounters(tt.args.ctx, tt.args.newValue); (err != nil) != tt.wantErr {
				t.Errorf("ReloadAllCounters() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemCashed_ReloadAllGauges(t *testing.T) {

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx      context.Context
		newValue map[string]float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			args:    args{newValue: make(map[string]float64)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			if err := m.ReloadAllGauges(tt.args.ctx, tt.args.newValue); (err != nil) != tt.wantErr {
				t.Errorf("ReloadAllGauges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemCashed_ReloadAllMetrics(t *testing.T) {

	var metricsWithoutErrors = []models.StorageMetrics{
		{MType: "counter", Name: "name", Delta: new(int64)},
		{MType: "gauge", Name: "name", Value: new(float64)},
	}

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx     context.Context
		metrics []models.StorageMetrics
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "Загрузка 2 метрик",
			args: args{
				ctx:     context.Background(),
				metrics: metricsWithoutErrors,
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "Загрузка метрики с ошибкой",
			args: args{
				ctx:     context.Background(),
				metrics: []models.StorageMetrics{{MType: "error_type"}},
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			got, err := m.ReloadAllMetrics(tt.args.ctx, tt.args.metrics)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReloadAllMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReloadAllMetrics() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemCashed_UpdateCounter(t *testing.T) {

	counterWithValue := make(map[string]int64)
	counterWithValue["name"] = 1

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx         context.Context
		metricsName string
		delta       int64
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantValue int64
		wantErr   bool
	}{
		{
			name:      "Counter не существует",
			fields:    fields{Counter: make(map[string]int64)},
			args:      args{ctx: context.Background(), metricsName: "name", delta: 2},
			wantValue: 2,
			wantErr:   false,
		},
		{
			name:      "Counter существует",
			fields:    fields{Counter: counterWithValue},
			args:      args{ctx: context.Background(), metricsName: "name", delta: 2},
			wantValue: 3,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			err := m.UpdateCounter(tt.args.ctx, tt.args.metricsName, tt.args.delta)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateCounter() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValue, m.Counter["name"])
		})
	}
}

func TestMemCashed_UpdateGauge(t *testing.T) {

	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	type args struct {
		ctx         context.Context
		metricsName string
		newValue    float64
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantValue float64
		wantErr   bool
	}{
		{
			name:      "Обновление Gauge",
			fields:    fields{Gauge: make(map[string]float64)},
			args:      args{ctx: context.Background(), metricsName: "name", newValue: 1},
			wantValue: 1,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemCashed{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			err := m.UpdateGauge(tt.args.ctx, tt.args.metricsName, tt.args.newValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateGauge() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValue, m.Gauge["name"])
		})
	}
}
