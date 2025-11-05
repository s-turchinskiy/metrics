// Package rsamiddleware Расшифровка тела запроса приватным ключом RSA в middleware
package rsamiddleware

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	error2 "github.com/s-turchinskiy/metrics/internal/common/error"
	rsautil "github.com/s-turchinskiy/metrics/internal/common/rsa"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"io"
	"net/http"
)

func RSADecrypt(privateKey *rsa.PrivateKey) func(next http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			if r.RequestURI != "/update" || r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			if privateKey == nil {
				next.ServeHTTP(w, r)
				return
			}

			if r.Body == nil {

				next.ServeHTTP(w, r)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Log.Debugw(error2.WrapError(fmt.Errorf("error read body")).Error())
				next.ServeHTTP(w, r)
				return
			}

			r.Body.Close()

			bodyBytes, err = rsautil.Decrypt(privateKey, bodyBytes)
			if err != nil {
				http.Error(w, "cannot decrypt body", http.StatusBadRequest)
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
