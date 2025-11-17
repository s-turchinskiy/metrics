package main

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/server/repository"
	"github.com/s-turchinskiy/metrics/internal/server/repository/memcashed"
	"github.com/s-turchinskiy/metrics/internal/server/repository/postgresql"
	closerutil "github.com/s-turchinskiy/metrics/internal/utils/closerutil"
	"log"
	_ "net/http/pprof"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/s-turchinskiy/metrics/internal/server/handlers"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
)

func init() {

	if err := logger.Initialize(); err != nil {
		log.Fatal(err)
	}

}

func main() {

	pathCert := "./cmd/server/certificate/cert.pem"
	pathRSAPrivateKey := "./cmd/server/certificate/rsa_private_key.pem"

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()
	closer := closerutil.New(20 * time.Second)

	err := godotenv.Load("./cmd/server/.env")
	if err != nil {
		logger.Log.Debugw("Error loading .env file", "error", err.Error())
	}

	if err = settings.GetSettings(); err != nil {
		logger.Log.Errorw("Get Settings error", "error", err.Error())
		log.Fatal(err)
	}

	var rep repository.Repository
	if settings.Settings.Store == settings.Database {

		rep, err = postgresql.Initialize(ctx, settings.Settings.Database.String(), settings.Settings.Database.DBName)
		if err != nil {
			logger.Log.Debugw("Connect to database error", "error", err.Error())
			log.Fatal(err)
		}
		closer.Add(rep.Close)

	} else {

		rep = &memcashed.MemCashed{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		}

	}

	metricsHandler := handlers.NewHandler(ctx, rep, settings.Settings.FileStoragePath, settings.Settings.AsynchronousWritingDataToFile)
	httpServer := handlers.NewHTTPServer(
		metricsHandler,
		settings.Settings.Address.String(),
		10*time.Second,
		10*time.Second,
		settings.Settings.RSAPrivateKey,
		settings.Settings.HashKey,
		settings.Settings.TrustedSubnetTyped,
	)
	closer.Add(httpServer.FuncShutdown(logger.Log))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		err = httpServer.Run(settings.Settings.EnableHTTPS, pathCert, pathRSAPrivateKey)
		if err != nil {

			logger.Log.Errorw("Server startup error", "error", err.Error())
			stop()
		}
	}()

	go func() {
		defer wg.Done()
		err = saveMetricsToFilePeriodically(ctx, metricsHandler)
		if err != nil {
			logger.Log.Errorw("Server startup error", "error", err.Error())
		}
	}()

	closer.Add(metricsHandler.Service.SaveMetricsToFile)

	<-ctx.Done()
	err = closer.Shutdown()

	wg.Wait()

	if err != nil {
		log.Fatal(err)
	}

}

func saveMetricsToFilePeriodically(ctx context.Context, h *handlers.MetricsHandler) error {

	if !settings.Settings.AsynchronousWritingDataToFile {
		return nil
	}

	ticker := time.NewTicker(time.Duration(settings.Settings.StoreInterval) * time.Second)
	for range ticker.C {

		err := h.Service.SaveMetricsToFile(ctx)
		if err != nil {
			logger.Log.Infoln("error", err.Error())
			return err
		}
	}
	return nil
}
