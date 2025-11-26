package main

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/server/grpchandlers"
	"github.com/s-turchinskiy/metrics/internal/server/repository"
	"github.com/s-turchinskiy/metrics/internal/server/repository/memcashed"
	"github.com/s-turchinskiy/metrics/internal/server/repository/postgresql"
	"github.com/s-turchinskiy/metrics/internal/server/service"
	"github.com/s-turchinskiy/metrics/internal/utils/closerutil"
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

	var serv *service.Service
	if settings.Settings.Store == settings.Database {

		retryStrategy := []time.Duration{
			0,
			2 * time.Second,
			5 * time.Second}

		serv = service.New(rep, retryStrategy, settings.Settings.FileStoragePath)

	} else {

		serv = service.New(rep, []time.Duration{0}, settings.Settings.FileStoragePath)

		if settings.Settings.Restore {
			err := serv.LoadMetricsFromFile(ctx)
			if err != nil {
				logger.Log.Errorw("LoadMetricsFromFile error", "error", err.Error())
				log.Fatal(err)
			}
		}
	}

	metricsHandler := handlers.NewHandler(ctx, serv, settings.Settings.AsynchronousWritingDataToFile)
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

	grpcServer := grpchandlers.New(
		serv,
		settings.Settings.PortGRPC,
		settings.Settings.HashKey,
		settings.Settings.RSAPrivateKey,
		settings.Settings.TrustedSubnetTyped,
	)

	closer.Add(grpcServer.Close)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		err = httpServer.Run(settings.Settings.EnableHTTPS, pathCert, pathRSAPrivateKey)
		if err != nil {
			logger.Log.Errorw("HTTP server startup error", "error", err.Error())
			stop()
		}
	}()

	go func() {
		defer wg.Done()
		err = grpcServer.Run()
		if err != nil {
			logger.Log.Errorw("gRPC server startup error", "error", err.Error())
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
