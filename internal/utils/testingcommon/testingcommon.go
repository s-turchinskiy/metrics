// Package testingcommon Общие процедуры тестирования
package testingcommon

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type Test struct {
	Name        string
	Method      string
	Address     string
	ContentType string
	Request     string
	Want        Want
}

type Want struct {
	ContentType string
	StatusCode  int
	Response    string
}

type TestPostGzip struct {
	Name         string
	ResponseCode int
	RequestBody  string
	ResponseBody string
}

func TestGzipCompression(t *testing.T, handler http.Handler, tests []TestPostGzip) {

	srv := httptest.NewServer(handler)
	defer srv.Close()

	for _, test := range tests {

		t.Run(test.Name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			zb := gzip.NewWriter(buf)
			_, err := zb.Write([]byte(test.RequestBody))
			require.NoError(t, err)
			err = zb.Close()
			require.NoError(t, err)
			r := httptest.NewRequest("POST", srv.URL, buf)
			r.RequestURI = ""
			r.Header.Set("Content-Encoding", "gzip")
			r.Header.Set("Accept-Encoding", "gzip")

			resp, err := http.DefaultClient.Do(r)
			require.NoError(t, err)
			require.Equal(t, test.ResponseCode, resp.StatusCode)

			defer resp.Body.Close()

			if test.ResponseBody != "" {
				zr, err := gzip.NewReader(resp.Body)
				require.NoError(t, err)

				b, err := io.ReadAll(zr)
				require.NoError(t, err)

				require.JSONEq(t, test.ResponseBody, string(b))
			}
		})
	}
}
