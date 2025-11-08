package main

import (
	"context"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric/httpresty"
	"github.com/s-turchinskiy/metrics/internal/common/closerutil"
	"github.com/s-turchinskiy/metrics/internal/common/hashutil"
	"log"
	"os/signal"
	"sync"
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

	cfg, err := config.ParseFlags()
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
		ServerAddress: "http://" + cfg.Addr.String(),
	}

	errorsCh := make(chan error)
	go closer.ProcessingErrorsChannel(errorsCh)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		go services.UpdateMetrics(ctx, metricsHandler, cfg.PollInterval, errorsCh)
	}()

	sender := httpresty.New(
		fmt.Sprintf("%s/update/", metricsHandler.ServerAddress),
		hashutil.СomputeHexadecimalSha256Hash,
		cfg.HashKey,
		cfg.RSAPublicKey,
	)
	go func() {
		defer wg.Done()

		reporter.ReportMetrics(
			ctx,
			metricsHandler,
			sender,
			cfg.ReportInterval,
			cfg.RateLimit,
			errorsCh)
	}()

	//go reporter.ReportMetricsBatch(metricsHandler, cfg.ReportInterval, errors)

	<-ctx.Done()
	err = closer.Shutdown()

	wg.Wait()

	log.Fatal(err)

}

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}
