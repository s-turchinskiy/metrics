package service

import (
	"context"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"github.com/s-turchinskiy/metrics/internal/server/repository/memcashed"
)

func ExampleService_UpdateTypedMetrics() {

	rep := &memcashed.MemCashed{
		Gauge:   map[string]float64{"someMetric": 1.23},
		Counter: map[string]int64{"someMetric": 2},
	}

	var value = 1.23
	var delta int64 = 2
	metrics := []models.StorageMetrics{
		{MType: "Gauge", Name: "someMetric", Value: &value},
		{MType: "Counter", Name: "someMetric", Delta: &delta},
	}

	s := New(rep, nil)
	res, err := s.UpdateTypedMetrics(context.Background(), metrics)
	fmt.Println(res, err)

	// Output:
	// 2, nil
}
