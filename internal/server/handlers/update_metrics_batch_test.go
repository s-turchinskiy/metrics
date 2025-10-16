package handlers

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/s-turchinskiy/metrics/internal/common/testingcommon"
	mocksrepository "github.com/s-turchinskiy/metrics/internal/server/repository/mock"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandler_UpdateMetricsBatch(t *testing.T) {

	address := "/updates"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := mocksrepository.NewMockRepository(ctrl)

	ctx1 := context.Background()
	var res1 int64 = 2
	mock.EXPECT().ReloadAllMetrics(ctx1, gomock.Any()).Return(res1, nil)

	ctx2 := context.Background()
	var res2 int64 = 0
	mock.EXPECT().ReloadAllMetrics(ctx2, gomock.Any()).Return(res2, fmt.Errorf("error"))

	tests := []test{
		{
			handler: NewHandler(ctx1, mock),
			ct: testingcommon.Test{
				Name:        "Успешно",
				Method:      http.MethodPost,
				Address:     address,
				ContentType: ContentTypeApplicationJSON,
				Request: `[
   						{
      						"id": "PauseTotalNs",
      						"type": "gauge",
      						"value": 128
  					 	},
   	 					{
      						"id": "PollCount",
      						"type": "counter",
      						"delta": 5
						}
						]`,
				Want: testingcommon.Want{
					StatusCode: http.StatusOK,
				}},
		},
		{
			handler: NewHandler(ctx2, mock),
			ct: testingcommon.Test{
				Name:        "Не успешно",
				Method:      http.MethodPost,
				Address:     address,
				ContentType: ContentTypeApplicationJSON,
				Request: `[
   						{
      						"id": "PauseTotalNs",
      						"type": "gauge",
      						"value": 128
  					 	}
						]`,
				Want: testingcommon.Want{
					StatusCode: http.StatusBadRequest,
				}},
		},
		{
			handler: NewHandler(ctx2, mock),
			ct: testingcommon.Test{
				Name:        "Неправильный json",
				Method:      http.MethodPost,
				Address:     address,
				ContentType: ContentTypeApplicationJSON,
				Request: `[
   						{
      						"id": "PauseTotalNs",
      						"type": "gauge",
  					 	}
						]`,
				Want: testingcommon.Want{
					StatusCode: http.StatusBadRequest,
				}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.ct.Name, func(t *testing.T) {
			r := httptest.NewRequest(tt.ct.Method, tt.ct.Address, strings.NewReader(tt.ct.Request))
			if tt.ct.ContentType != "" {
				r.Header.Set("Content-Type", tt.ct.ContentType)
			}
			w := httptest.NewRecorder()
			tt.handler.UpdateMetricsBatch(w, r)

			result := w.Result()
			defer result.Body.Close()

			assert.Equal(t, tt.ct.Want.StatusCode, result.StatusCode)

		})
	}
}
