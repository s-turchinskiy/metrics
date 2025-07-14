package main

import (
	"github.com/s-turchinskiy/metrics/internal/testingcommon"
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

func TestMetricsHandler_UpdateMetricJSON(t *testing.T) {

	settings = ProgramSettings{Restore: false, asynchronousWritingDataToFile: true}
	h := &MetricsHandler{storage: &MetricsStorage{
		Gauge:   map[string]float64{"someMetric": 1.23},
		Counter: make(map[string]int64)}}
	handler := http.HandlerFunc(gzipMiddleware(h.UpdateMetricJSON))

	test1 := testingcommon.TestPostGzip{Name: "Gauge отправка корректного значения",
		ResponseCode: 200,
		RequestBody: `{
    	"id" : "someMetric",
		"type" : "gauge",
		"value" : 1.25
		}`,

		ResponseBody: `{
    	"id" : "someMetric",
		"type" : "gauge",
		"value" : 1.25
		}`,
	}

	test2 := testingcommon.TestPostGzip{Name: "Counter отправка без указания delta",
		ResponseCode: http.StatusBadRequest,
		RequestBody: `{
		"id" : "someMetric",
		"type" : "counter",
		"value" : 2
		}`,
	}

	test3 := testingcommon.TestPostGzip{Name: "Counter отправка некорретного delta",
		ResponseCode: http.StatusInternalServerError,
		RequestBody: `{
		"id" : "someMetric",
		"type" : "counter",
		"delta" : 1.23
		}`,
	}

	test4 := testingcommon.TestPostGzip{Name: "Counter отправка первого значения",
		ResponseCode: 200,
		RequestBody: `{
		"id" : "someMetric",
		"type" : "counter",
		"delta" : 3
		}`,

		ResponseBody: `{
		"id" : "someMetric",
		"type" :"counter",
		"delta" : 3
		}`,
	}

	test5 := testingcommon.TestPostGzip{Name: "Counter отправка второго значения",
		ResponseCode: 200,
		RequestBody: `{
		    	"id" : "someMetric",
				"type" : "counter",
				"delta" : 5
				}`,

		ResponseBody: `{
		    	"id" : "someMetric",
				"type" :"counter",
				"delta" : 8
				}`,
	}

	tests := []testingcommon.TestPostGzip{test1, test2, test3, test4, test5}
	testingcommon.TestGzipCompression(t, handler, tests)
}

func TestMetricsHandler_GetTypedMetric(t *testing.T) {

	settings = ProgramSettings{Restore: false, asynchronousWritingDataToFile: true}

	h := &MetricsHandler{storage: &MetricsStorage{
		Gauge:   map[string]float64{"someMetric": 1.23},
		Counter: make(map[string]int64)}}
	handler := http.HandlerFunc(gzipMiddleware(h.GetTypedMetric))

	test1 := testingcommon.TestPostGzip{Name: "Gauge проверка присутствующего значения",
		ResponseCode: 200,
		RequestBody: `{
    	"id" : "someMetric",
		"type" : "gauge"
		}`,

		ResponseBody: `{
    	"id" : "someMetric",
		"type" : "gauge",
		"value" : 1.23
		}`,
	}

	test2 := testingcommon.TestPostGzip{Name: "Gauge проверка отсутствующего значения",
		ResponseCode: 200,
		RequestBody: `{
    	"id" : "someMetric1",
		"type" : "gauge"
		}`,

		ResponseBody: `{
    	"id" : "someMetric1",
		"type" : "gauge",
		"value" : 0
		}`,
	}

	tests := []testingcommon.TestPostGzip{test1, test2}
	testingcommon.TestGzipCompression(t, handler, tests)

}
