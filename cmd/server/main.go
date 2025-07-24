package main

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/s-turchinskiy/metrics/internal/server"
	"github.com/s-turchinskiy/metrics/internal/server/handlers"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"go.uber.org/zap"
	"log"
	"net/http"
	"time"
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

	metricsHandler := handlers.NewHandler(ctx)

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

	router := server.Router(h)

	logger.Log.Info("Running server", zap.String("address", settings.Settings.Address.String()))

	return http.ListenAndServe(settings.Settings.Address.String(), router)
}
