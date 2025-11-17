package handlers

import (
	"context"
	"crypto/rsa"
	"errors"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/utils/rsautil"
	"go.uber.org/zap"
	"net"
	"net/http"
	"os"
	"time"
)

type HTTPServer struct {
	*http.Server
}

func NewHTTPServer(
	handler *MetricsHandler,
	addr string, readTimeout,
	writeTimeout time.Duration,
	rsaPrivateKey *rsa.PrivateKey,
	hashKey string,
	trustedSubnet *net.IPNet) *HTTPServer {

	server := &http.Server{
		Addr:         addr,
		Handler:      Router(handler, rsaPrivateKey, hashKey, trustedSubnet),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	return &HTTPServer{server}

}

func (httpServer *HTTPServer) Run(enableHTTPS bool, pathCert, pathRSAPrivateKey string) error {

	var err error

	if enableHTTPS {

		if _, err = os.Stat(pathCert); err != nil && errors.Is(err, os.ErrNotExist) {
			err = rsautil.GenerateCertificateHTTPS(pathCert, pathRSAPrivateKey)
			if err != nil {
				return err
			}
		}

		err = httpServer.ListenAndServeTLS(pathCert, pathRSAPrivateKey)

	} else {

		err = httpServer.ListenAndServe()
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {

		logger.Log.Errorw("Server startup error", "error", err.Error())
		return err
	}

	return nil

}

// WrapShutdown logger.Log.Infow - это по факту не работает, s.base.Core().Enabled(lvl) начинает возвращать false, но причем только тут
func (httpServer *HTTPServer) WrapShutdown(ctx context.Context) error {

	err := httpServer.Shutdown(ctx)
	if err != nil {
		logger.Log.Infow("HTTP server stopped with error", zap.String("error", err.Error()))
	} else {
		logger.Log.Infow("HTTP server stopped")
	}
	return err
}

func (httpServer *HTTPServer) FuncShutdown(zaplog *zap.SugaredLogger) func(ctx context.Context) error {

	return func(ctx context.Context) error {
		err := httpServer.Shutdown(ctx)
		if err != nil {
			zaplog.Infow("HTTP server stopped with error", zap.String("error", err.Error()))
		} else {
			zaplog.Infow("HTTP server stopped")
		}
		return err
	}
}
