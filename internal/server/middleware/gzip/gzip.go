package gzip

import (
	"compress/gzip"
	"io"
	"net/http"
	"slices"
	"strings"
)

var contentTypeForCompress = []string{"application/json", "text/html"}

type compressWriter struct {
	http.ResponseWriter
	zw            *gzip.Writer
	statusCodeSet bool
}

func GzipMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")

		if supportsGzip {
			cw := newCompressWriter(w)
			w = cw
			defer cw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		next.ServeHTTP(w, r)

	})
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		ResponseWriter: w,
		zw:             gzip.NewWriter(w),
	}
}

func (c *compressWriter) WriteHeader(statusCode int) {

	if statusCode < 300 {
		c.Header().Set("Content-Encoding", "gzip")
	}
	c.ResponseWriter.WriteHeader(statusCode)
	c.statusCodeSet = true
}

func (c *compressWriter) Write(p []byte) (int, error) {

	if !c.statusCodeSet {
		c.Header().Set("Content-Encoding", "gzip")
	}

	ContentType := c.Header().Get("Content-Type")
	supportsContentType := slices.Contains(contentTypeForCompress, ContentType)
	if supportsContentType {
		return c.zw.Write(p)
	}

	return c.ResponseWriter.Write(p)
}

func (c *compressWriter) Close() error {
	return c.zw.Close()
}

type compressReader struct {
	io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		ReadCloser: r,
		zr:         zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.ReadCloser.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
