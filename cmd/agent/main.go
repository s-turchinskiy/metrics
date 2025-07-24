package main

import (
	"github.com/joho/godotenv"
	"github.com/s-turchinskiy/metrics/cmd/agent/config"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/repositories"
	"github.com/s-turchinskiy/metrics/internal/agent/services"
	"log"
)

func main() {

	if err := logger.Initialize(); err != nil {
		log.Fatal(err)
	}

	err := godotenv.Load("./cmd/agent/.env")
	if err != nil {
		logger.Log.Debugw("Error loading .env file", "error", err.Error())
	}

	addr := config.NetAddress{Host: "localhost", Port: 8080}
	config.ParseFlags(&addr)

	metricsHandler := &services.MetricsHandler{
		Storage: &repositories.MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		},
		ServerAddress: "http://" + addr.String(),
	}

	errorsChan := make(chan error)

	go services.UpdateMetrics(metricsHandler, errorsChan)
	go services.ReportMetrics(metricsHandler, errorsChan)
	//go services.ReportMetricsBatch(metricsHandler, errors)

	err = <-errorsChan
	log.Fatal(err)

}
