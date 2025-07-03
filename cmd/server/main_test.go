package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type want struct {
	contentType string
	statusCode  int
	response    string
	storage     MetricsUpdater
}

type test struct {
	name    string
	method  string
	request string
	storage MetricsUpdater
	want    want
}

func TestMetricsHandler_UpdateMetric(t *testing.T) {

	tests := []test{{
		name:    "успешное добавление метрики",
		method:  http.MethodPost,
		request: "/update/gauge/someMetric/1.1",
		storage: &MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64)},
		want: want{
			contentType: "text/html",
			statusCode:  200,
			storage: &MetricsStorage{
				Gauge:   map[string]float64{"someMetric": 1.1},
				Counter: make(map[string]int64),
			},
		},
	},
		{
			name:    "метод Get запрещен",
			method:  http.MethodGet,
			request: "/update/gauge/someMetric/1.1",
			storage: &MetricsStorage{
				Gauge:   make(map[string]float64),
				Counter: make(map[string]int64)},
			want: want{
				contentType: "text/html",
				statusCode:  http.StatusMethodNotAllowed,
				storage: &MetricsStorage{
					Gauge:   make(map[string]float64),
					Counter: make(map[string]int64),
				},
			},
		},
		{
			name:    "Значение не float64",
			method:  http.MethodPost,
			request: "/update/gauge/someMetric/bad",
			storage: &MetricsStorage{
				Gauge:   make(map[string]float64),
				Counter: make(map[string]int64)},
			want: want{
				contentType: "text/html",
				statusCode:  http.StatusBadRequest,
				storage: &MetricsStorage{
					Gauge:   make(map[string]float64),
					Counter: make(map[string]int64),
				},
				response: "MetricsValue = bad, error: strconv.ParseFloat: parsing \"bad\": invalid syntax",
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.request, nil)
			w := httptest.NewRecorder()
			h := &MetricsHandler{
				storage: tt.storage,
			}
			h.UpdateMetric(w, request)

			result := w.Result()

			defer result.Body.Close()
			resBody, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.statusCode, result.StatusCode)

			assert.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.response, string(resBody))
			//assert.InDeltaMapValues(t, tt.want.storage.(*MetricsStorage).Gauge, tt.storage.(*MetricsStorage).Gauge, 64)

			eq := reflect.DeepEqual(tt.want.storage.(*MetricsStorage).Gauge, tt.storage.(*MetricsStorage).Gauge)
			if !eq {
				t.Error("MetricsStorage are unequal.")
			}

		})
	}
}

func TestMetricsHandler_GetMetric(t *testing.T) {

	tests := []test{{
		name:    "запрос отсутствующей метрики",
		method:  http.MethodGet,
		request: "/value/gauge/someMetric",
		storage: &MetricsStorage{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64)},
		want: want{
			contentType: "text/html",
			statusCode:  http.StatusNotFound,
			response:    "not found",
		},
	},
		{
			name:    "запрос присутсвующей метрики",
			method:  http.MethodGet,
			request: "/value/gauge/someMetric",
			storage: &MetricsStorage{
				Gauge:   map[string]float64{"someMetric": 1.23},
				Counter: make(map[string]int64)},
			want: want{
				contentType: "text/html",
				statusCode:  http.StatusOK,
				response:    "1.23",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.request, nil)
			w := httptest.NewRecorder()
			h := &MetricsHandler{
				storage: tt.storage,
			}
			h.GetMetric(w, request)

			result := w.Result()

			defer result.Body.Close()
			resBody, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.statusCode, result.StatusCode)

			assert.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.response, string(resBody))

		})
	}
}
