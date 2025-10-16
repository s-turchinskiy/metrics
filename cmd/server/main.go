package main

import (
	"context"
	"github.com/s-turchinskiy/metrics/internal/server/repository"
	"github.com/s-turchinskiy/metrics/internal/server/repository/memcashed"
	"github.com/s-turchinskiy/metrics/internal/server/repository/postgresql"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

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

	ctx := context.Background()

	err := godotenv.Load("./cmd/server/.env")
	if err != nil {
		logger.Log.Debugw("Error loading .env file", "error", err.Error())
	}

	if err := settings.GetSettings(); err != nil {
		logger.Log.Errorw("Get Settings error", "error", err.Error())
		log.Fatal(err)
	}

	var rep repository.Repository
	if settings.Settings.Store == settings.Database {

		rep, err = postgresql.Initialize(ctx)
		if err != nil {
			logger.Log.Debugw("Connect to database error", "error", err.Error())
			log.Fatal(err)
		}

	} else {

		rep = &memcashed.MemCashed{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		}

	}

	metricsHandler := handlers.NewHandler(ctx, rep)

	errors := make(chan error)

	go func() {
		err := run(metricsHandler)
		if err != nil {

			logger.Log.Errorw("Server startup error", "error", err.Error())
			errors <- err
			return
		}
	}()

	go saveMetricsToFilePeriodically(ctx, metricsHandler, errors)

	err = <-errors
	metricsHandler.Service.SaveMetricsToFile(ctx)
	logger.Log.Infow("error, server stopped", "error", err.Error())
	log.Fatal(err)
}

func saveMetricsToFilePeriodically(ctx context.Context, h *handlers.MetricsHandler, errors chan error) {

	if !settings.Settings.AsynchronousWritingDataToFile {
		return
	}

	ticker := time.NewTicker(time.Duration(settings.Settings.StoreInterval) * time.Second)
	for range ticker.C {

		err := h.Service.SaveMetricsToFile(ctx)
		if err != nil {
			logger.Log.Infoln("error", err.Error())
			errors <- err
			return
		}
	}
}

func run(h *handlers.MetricsHandler) error {

	router := handlers.Router(h)

	logger.Log.Info("Running server", zap.String("address", settings.Settings.Address.String()))

	return http.ListenAndServe(settings.Settings.Address.String(), router)
}
