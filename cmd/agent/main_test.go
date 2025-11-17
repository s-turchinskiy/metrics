package main

import (
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/agent/repositories"
	"github.com/s-turchinskiy/metrics/internal/agent/services"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric/httpresty"
	"testing"
	"time"
)

func BenchmarkAll(b *testing.B) {

	storage := &repositories.MetricsStorage{
		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}

	sender := httpresty.New(
		fmt.Sprintf("%s/update/", "http://notfound"),
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		metrics, err := services.GetMetrics(time.Nanosecond)
		if err != nil {
			b.Fatal(err)
		}
		err = storage.UpdateMetrics(metrics)
		if err != nil {
			b.Fatal(err)
		}

		metricsStorage, err := storage.GetMetrics()
		if err != nil {
			b.Fatal(err)
		}

		for _, metric := range metricsStorage {
			sender.Send(metric)
		}
	}

}
