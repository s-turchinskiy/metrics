package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/s-turchinskiy/metrics/internal/server/handlers"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/gzip"
	"github.com/s-turchinskiy/metrics/internal/server/repository/memcashed"
	"github.com/s-turchinskiy/metrics/internal/server/service"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"github.com/s-turchinskiy/metrics/internal/utils/testingcommon"
)

type want struct {
	contentType string
	statusCode  int
	response    string
	storage     service.MetricsUpdater
}

type test struct {
	name    string
	method  string
	request string
	storage service.MetricsUpdater
	want    want
}

func EmptyService() *service.Service {
	return &service.Service{
		Repository: &memcashed.MemCashed{
			Gauge:   make(map[string]float64),
			Counter: make(map[string]int64),
		},
	}
}

func TestMetricsHandler_UpdateMetric(t *testing.T) {

	tests := []test{
		{
			name:    "успешное добавление метрики gauge",
			method:  http.MethodPost,
			request: "/update/gauge/someMetric/1.1",
			storage: EmptyService(),
			want: want{
				contentType: handlers.ContentTypeTextHTML,
				statusCode:  200,
				storage: &service.Service{
					Repository: &memcashed.MemCashed{
						Gauge:   map[string]float64{"someMetric": 1.1},
						Counter: make(map[string]int64),
					},
				},
			},
		},
		{
			name:    "успешное добавление метрики counter",
			method:  http.MethodPost,
			request: "/update/counter/someMetric/2",
			storage: EmptyService(),
			want: want{
				contentType: handlers.ContentTypeTextHTML,
				statusCode:  200,
				storage: &service.Service{
					Repository: &memcashed.MemCashed{
						Gauge:   make(map[string]float64),
						Counter: map[string]int64{"someMetric": 2},
					},
				},
			},
		},
		{
			name:    "метод Get запрещен",
			method:  http.MethodGet,
			request: "/update/gauge/someMetric/1.1",
			storage: EmptyService(),
			want: want{
				contentType: handlers.ContentTypeTextHTML,
				statusCode:  http.StatusMethodNotAllowed,
				storage:     EmptyService(),
			},
		},
		{
			name:    "Значение не float64",
			method:  http.MethodPost,
			request: "/update/gauge/someMetric/bad",
			storage: EmptyService(),
			want: want{
				contentType: handlers.ContentTypeTextHTML,
				statusCode:  http.StatusBadRequest,
				storage:     EmptyService(),
				response:    "MetricsValue = bad, error: strconv.ParseFloat: parsing \"bad\": invalid syntax",
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.request, nil)
			w := httptest.NewRecorder()
			h := &handlers.MetricsHandler{
				Service: tt.storage,
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
			//assert.InDeltaMapValues(t, tt.want.storage.(*Service).Gauge, tt.storage.(*Service).Gauge, 64)

			eq := reflect.DeepEqual(tt.want.storage.(*service.Service).Repository, tt.storage.(*service.Service).Repository)
			if !eq {
				t.Error("Service are unequal.")
			}

		})
	}
}

func TestMetricsHandler_GetMetric(t *testing.T) {

	tests := []test{
		{
			name:    "запрос отсутствующей метрики",
			method:  http.MethodGet,
			request: "/value/gauge/someMetric",
			storage: EmptyService(),
			want: want{
				contentType: handlers.ContentTypeTextHTML,
				statusCode:  http.StatusNotFound,
				response:    "not found",
			},
		},
		{
			name:    "запрос присутствующей метрики gauge",
			method:  http.MethodGet,
			request: "/value/gauge/someMetric",
			storage: &service.Service{
				Repository: &memcashed.MemCashed{
					Gauge:   map[string]float64{"someMetric": 1.23},
					Counter: make(map[string]int64),
				},
			},
			want: want{
				contentType: handlers.ContentTypeTextHTML,
				statusCode:  http.StatusOK,
				response:    "1.23",
			},
		},
		{
			name:    "запрос присутствующей метрики counter",
			method:  http.MethodGet,
			request: "/value/counter/someMetric",
			storage: &service.Service{
				Repository: &memcashed.MemCashed{
					Gauge:   make(map[string]float64),
					Counter: map[string]int64{"someMetric": 2},
				},
			},
			want: want{
				contentType: handlers.ContentTypeTextHTML,
				statusCode:  http.StatusOK,
				response:    "2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.request, nil)
			w := httptest.NewRecorder()
			h := &handlers.MetricsHandler{
				Service: tt.storage,
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

	settings.Settings = settings.ProgramSettings{Restore: false, AsynchronousWritingDataToFile: true}

	rep := &memcashed.MemCashed{
		Gauge:   map[string]float64{"someMetric": 1.23},
		Counter: make(map[string]int64),
	}

	h := &handlers.MetricsHandler{Service: service.New(rep, nil, "")}

	handler := gzip.GzipMiddleware(http.HandlerFunc(h.UpdateMetricJSON))

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

	test6 := testingcommon.TestPostGzip{Name: "Битый json",
		ResponseCode: http.StatusInternalServerError,
		RequestBody: `{
		    	"id" : "someMetric",
				"type" : "counter",
				"delta" : 5 fgfg,,,,
				}`,

		ResponseBody: ``,
	}

	tests := []testingcommon.TestPostGzip{test1, test2, test3, test4, test5, test6}
	testingcommon.TestGzipCompression(t, handler, tests)
}

func TestMetricsHandler_GetTypedMetric(t *testing.T) {

	settings.Settings = settings.ProgramSettings{Restore: false, AsynchronousWritingDataToFile: true}

	rep := &memcashed.MemCashed{
		Gauge:   map[string]float64{"someMetric": 1.23},
		Counter: map[string]int64{"someMetric": 2},
	}

	h := &handlers.MetricsHandler{Service: service.New(rep, nil, "")}

	//handler := http.HandlerFunc(gzipMiddleware(h.GetTypedMetric))
	handler := gzip.GzipMiddleware(http.HandlerFunc(h.GetTypedMetric))

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

	test3 := testingcommon.TestPostGzip{Name: "Counter проверка присутствующего значения",
		ResponseCode: 200,
		RequestBody: `{
    	"id" : "someMetric",
		"type" : "counter"
		}`,

		ResponseBody: `{
    	"id" : "someMetric",
		"type" : "counter",
		"delta" : 2
		}`,
	}

	test4 := testingcommon.TestPostGzip{Name: "Битый json",
		ResponseCode: http.StatusInternalServerError,
		RequestBody: `{
    	"id" : "someMetric",
		"type" : "counter"vsfcgvd
		}`,

		ResponseBody: ``,
	}

	tests := []testingcommon.TestPostGzip{test1, test2, test3, test4}
	testingcommon.TestGzipCompression(t, handler, tests)

}

func TestInspectDatabase(t *testing.T) {

	/*settings.GetSettings()
	settings.Settings.Store = settings.Database
	ctx := context.Background()
	dbconn := sqlx.MustOpen("pgx", settings.Settings.Database.String())
	if err := dbconn.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	suite := suite.Suite{}
	id := "PopulateCounter" + strconv.Itoa(rand.Intn(256*256*256))

	httpc := resty.New().SetBaseURL(settings.Settings.Address.String())

	suite.Run("populate counter", func() {
		req := httpc.R().
			SetHeader("Content-Type", "application/json")

		var value int64
		resp, err := req.
			SetBody(
				&models.Metrics{
					ID:    id,
					MType: "counter",
					Delta: &value,
				}).
			Post("update/")

		dumpErr := suite.Assert().NoError(err,
			"Ошибка при попытке сделать запрос с обновлением counter")
		dumpErr = dumpErr && suite.Assert().Equalf(http.StatusOK, resp.StatusCode(),
			"Несоответствие статус кода ответа ожидаемому в хендлере %q: %q ", req.Method, req.URL)
		dumpErr = dumpErr && suite.Assert().NoError(err, "Ошибка при попытке сделать запрос для сокращения URL")

		if !dumpErr {
			log.Fatal(dumpErr)
		}
	})

	suite.Run("delay", func() {
		timeutil.Sleep(5 * timeutil.Second)
	})

	suite.Run("inspect", func() {
		suite.Require().NotNil(dbconn,
			"Невозможно проинспектировать базу данных, нет подключения")

		tables, err := fetchTables(dbconn)
		suite.Require().NoError(err,
			"Ошибка получения списка таблиц базы данных")
		suite.Require().NotEmpty(tables,
			"Не найдено ни одной пользовательской таблицы в БД")

	})*/
}

func fetchTables(dbconn *sqlx.DB) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	query := `
		SELECT
			table_schema || '.' || table_name
		FROM information_schema.tables
		WHERE
			table_schema NOT IN ('pg_catalog', 'information_schema')
	`

	rows, err := dbconn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("не удалось выполнить запрос листинга таблиц: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tablename string
		if err := rows.Scan(&tablename); err != nil {
			return nil, fmt.Errorf("не удалось получить строку результата: %w", err)
		}
		tables = append(tables, tablename)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка обработки курсора базы данных: %w", err)
	}
	return tables, nil
}
