package main

import (
	"context"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/agent/repositories"
	"github.com/s-turchinskiy/metrics/internal/agent/sender"
	"github.com/s-turchinskiy/metrics/internal/agent/sender/grpcsender"
	"github.com/s-turchinskiy/metrics/internal/agent/sender/httpresty"
	"github.com/s-turchinskiy/metrics/internal/utils/closerutil"
	"github.com/s-turchinskiy/metrics/internal/utils/errutil"
	"github.com/s-turchinskiy/metrics/internal/utils/hashutil"
	"log"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/s-turchinskiy/metrics/cmd/agent/config"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/reporter"
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

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()
	closer := closerutil.New(20 * time.Second)

	storage := &repositories.MetricsStorage{
		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}

	var sender sender.MetricSender
	switch cfg.SendingVia {
	case config.HTTP:
		sender = httpresty.New(
			cfg.URL,
			httpresty.WithHash(cfg.HashKey, hashutil.СomputeHexadecimalSha256Hash),
			httpresty.WithRsaPublicKey(cfg.RSAPublicKey),
			httpresty.WithRealIP(cfg.LocalIP),
		)
	case config.GRPC:
		sender = grpcsender.New(
			strconv.Itoa(cfg.Addr.Port),
			grpcsender.WithHash(cfg.HashKey, hashutil.СomputeHexadecimalSha256Hash),
			grpcsender.WithRsaPublicKey(cfg.RSAPublicKey),
			grpcsender.WithRealIP(cfg.LocalIP),
		)

		closer.Add(sender.Close)
	default:
		err = errutil.WrapError(fmt.Errorf("cfg.SendingVia is unklown, value = %d", cfg.SendingVia))
		log.Fatal(err)
	}

	report := reporter.New(storage, sender, cfg.ReportInterval, cfg.RateLimit)
	service := services.New(storage, report, cfg.Addr.String(), cfg.PollInterval)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		err = service.Run(ctx)
		if err != nil {
			logger.Log.Info(err)
			//closer.ProcessingErrors(err)
			stop()
		}
	}()

	<-ctx.Done()
	err = closer.Shutdown()

	wg.Wait()

	if err != nil {
		log.Fatal(err)
	}

}

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}
