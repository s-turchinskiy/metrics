package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"github.com/s-turchinskiy/metrics/cmd/agent/config"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/reporter"
	"github.com/s-turchinskiy/metrics/internal/agent/repositories"
	"github.com/s-turchinskiy/metrics/internal/agent/services"
)

// go run -ldflags "-X main.buildVersion=v1.0.1 -X main.buildDate=20.10.2025 -X main.buildCommit=Comment"
var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {

	printBuildInfo()

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

	doneCh := make(chan struct{})
	defer close(doneCh)

	go services.UpdateMetrics(metricsHandler, errorsChan, doneCh)
	go reporter.ReportMetrics(metricsHandler, errorsChan, doneCh)
	//go reporter.ReportMetricsBatch(metricsHandler, errors)

	err = <-errorsChan
	log.Fatal(err)

}

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}
