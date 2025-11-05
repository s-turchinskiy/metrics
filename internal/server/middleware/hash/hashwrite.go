package hash

import (
	"bytes"
	"github.com/s-turchinskiy/metrics/internal/common/hashutil"
	"net/http"

	"github.com/s-turchinskiy/metrics/internal/server/settings"
)

type hashingResponseWriter struct {
	http.ResponseWriter
	body          *bytes.Buffer
	statusCodeSet bool
}

func (hw *hashingResponseWriter) WriteHeader(statusCode int) {

	hw.ResponseWriter.WriteHeader(statusCode)
	hw.statusCodeSet = true
}

func (hw *hashingResponseWriter) Write(b []byte) (int, error) {

	hw.body.Write(b)

	if !hw.statusCodeSet && hw.body.Len() > 0 {
		hash := hashutil.СomputeHexadecimalSha256Hash(settings.Settings.HashKey, hw.body.Bytes())
		hw.Header().Set("HashSHA256", hash)
	}
	return hw.ResponseWriter.Write(b)
}

func HashWriteMiddleware(next http.Handler) http.Handler {
	hashFn := func(w http.ResponseWriter, r *http.Request) {

		hashW := &hashingResponseWriter{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
		}

		next.ServeHTTP(hashW, r)

	}

	return http.HandlerFunc(hashFn)
}
