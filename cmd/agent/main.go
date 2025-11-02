package main

import (
	"context"
	"fmt"
	closerutil "github.com/s-turchinskiy/metrics/internal/common/closer"
	"log"
	"os/signal"
	"syscall"
	"time"

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

	err = config.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()
	closer := closerutil.New(20 * time.Second)

	metricsHandler := &services.MetricsHandler{
		Storage: &repositories.MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		},
		ServerAddress: "http://" + config.Config.Addr.String(),
	}

	errorsCh := make(chan error)
	go closer.ProcessingErrorsChannel(errorsCh)

	doneCh := make(chan struct{})
	defer close(doneCh)

	go services.UpdateMetrics(ctx, metricsHandler, errorsCh, doneCh)
	go reporter.ReportMetrics(ctx, metricsHandler, errorsCh, doneCh, config.Config.RSAPublicKey)
	//go reporter.ReportMetricsBatch(metricsHandler, errors)

	<-ctx.Done()
	err = closer.Shutdown()

	log.Fatal(err)

}

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}
