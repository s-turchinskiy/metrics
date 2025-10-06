package hash

import (
	"bytes"
	"crypto/hmac"
	"encoding/hex"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/common"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"io"
	"net/http"
)

func HashReadMiddleware(next http.Handler) http.Handler {
	hashFn := func(w http.ResponseWriter, r *http.Request) {

		if settings.Settings.HashKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		requestHexadecimalHash := r.Header.Get("HashSHA256")
		if requestHexadecimalHash == "" {
			next.ServeHTTP(w, r)
			return
		}

		requestHash, err := hex.DecodeString(requestHexadecimalHash)
		if err != nil {
			http.Error(w, "Error decode request hash", http.StatusBadRequest)
			return
		}

		if r.Body == nil {

			next.ServeHTTP(w, r)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Log.Debugw(common.WrapError(fmt.Errorf("error read body")).Error())
			next.ServeHTTP(w, r)
			return
		}

		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		expectedHash := common.СomputeSha256Hash(settings.Settings.HashKey, bodyBytes)

		if !hmac.Equal(requestHash, expectedHash) {
			http.Error(w, "Invalid request hash", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(hashFn)
}
