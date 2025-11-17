package hash

import (
	"bytes"
	"github.com/s-turchinskiy/metrics/internal/utils/hashutil"
	"net/http"
)

type hashingResponseWriter struct {
	http.ResponseWriter
	body          *bytes.Buffer
	statusCodeSet bool
	hashKey       string
}

func (hw *hashingResponseWriter) WriteHeader(statusCode int) {

	hw.ResponseWriter.WriteHeader(statusCode)
	hw.statusCodeSet = true
}

func (hw *hashingResponseWriter) Write(b []byte) (int, error) {

	hw.body.Write(b)

	if !hw.statusCodeSet && hw.body.Len() > 0 {
		hash := hashutil.СomputeHexadecimalSha256Hash(hw.hashKey, hw.body.Bytes())
		hw.Header().Set("HashSHA256", hash)
	}
	return hw.ResponseWriter.Write(b)
}

func HashWriteMiddleware(hashKey string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hashFn := func(w http.ResponseWriter, r *http.Request) {

			hashW := &hashingResponseWriter{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				hashKey:        hashKey,
			}

			next.ServeHTTP(hashW, r)

		}

		return http.HandlerFunc(hashFn)
	}
}
