package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/s-turchinskiy/metrics/internal/agent/repositories"
	"github.com/s-turchinskiy/metrics/internal/agent/services"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric/httpresty"
	"github.com/s-turchinskiy/metrics/internal/common"
)

func BenchmarkAll(b *testing.B) {

	h := &services.MetricsHandler{
		Storage: &repositories.MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		},
		ServerAddress: "http://notfound",
	}

	sender := httpresty.New(
		fmt.Sprintf("%s/update/", h.ServerAddress),
		common.СomputeHexadecimalSha256Hash,
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		metrics, err := services.GetMetrics(time.Nanosecond)
		if err != nil {
			b.Fatal(err)
		}
		err = h.Storage.UpdateMetrics(metrics)
		if err != nil {
			b.Fatal(err)
		}

		metricsStorage, err := h.Storage.GetMetrics()
		if err != nil {
			b.Fatal(err)
		}

		for _, metric := range metricsStorage {
			sender.Send(metric)
		}
	}

}
