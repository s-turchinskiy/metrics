package main

import (
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/repositories"
	"github.com/s-turchinskiy/metrics/internal/agent/services"
	"log"
)

func main() {

	if err := logger.Initialize(); err != nil {
		log.Fatal(err)
	}

	addr := services.NetAddress{Host: "localhost", Port: 8080}
	parseFlags(&addr)

	metricsHandler := &services.MetricsHandler{
		Storage: &repositories.MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		},
		ServerAddress: "http://" + addr.String(),
	}

	errors := make(chan error)

	go services.UpdateMetrics(metricsHandler, errors)
	//go services.ReportMetrics(metricsHandler, errors)
	go services.ReportMetricsBatch(metricsHandler, errors)

	err := <-errors
	log.Fatal(err)

}
