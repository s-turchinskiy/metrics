package handlers

import (
	"context"
	"errors"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	rsautil "github.com/s-turchinskiy/metrics/internal/common/rsautil"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

func NewHTTPServer(handler *MetricsHandler, addr string, readTimeout, writeTimeout time.Duration) *http.Server {

	server := &http.Server{
		Addr:         addr,
		Handler:      Router(handler),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	return server

}

func RunHTTPServer(server *http.Server, enableHTTPS bool, pathCert, pathRSAPrivateKey string) error {

	var err error

	if enableHTTPS {

		if _, err = os.Stat(pathCert); err != nil && errors.Is(err, os.ErrNotExist) {
			err = rsautil.GenerateCertificateHTTPS(pathCert, pathRSAPrivateKey)
			if err != nil {
				return err
			}
		}

		err = server.ListenAndServeTLS(pathCert, pathRSAPrivateKey)

	} else {

		err = server.ListenAndServe()
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {

		logger.Log.Errorw("Server startup error", "error", err.Error())
		return err
	}

	return nil

}

func FuncHTTPServerShutdown(httpServer *http.Server) func(ctx context.Context) error {

	return func(ctx context.Context) error {
		err := httpServer.Shutdown(ctx)
		if err != nil {
			logger.Log.Infow("HTTP server stopped with error", zap.String("error", err.Error()))
		} else {
			logger.Log.Infow("HTTP server stopped")
		}
		return err
	}
}
