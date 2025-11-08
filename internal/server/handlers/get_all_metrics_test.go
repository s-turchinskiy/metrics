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

func TestMetricsHandler_GetAllMetrics(t *testing.T) {

	address := "/"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := mocksrepository.NewMockRepository(ctrl)

	ctx1 := context.Background()
	mock.EXPECT().GetAllGauges(ctx1).Return(make(map[string]float64), nil)
	mock.EXPECT().GetAllCounters(ctx1).Return(make(map[string]int64), nil)

	ctx2 := context.Background()
	mock.EXPECT().GetAllGauges(ctx2).Return(make(map[string]float64), nil)
	mock.EXPECT().GetAllCounters(ctx2).Return(make(map[string]int64), fmt.Errorf("error"))

	tests := []test{
		{
			handler: NewHandler(ctx1, mock, "", true),
			ct: testingcommon.Test{
				Name:        "Успешно",
				Method:      http.MethodGet,
				Address:     address,
				ContentType: ContentTypeTextPlain,
				Want: testingcommon.Want{
					StatusCode: http.StatusOK,
				}},
		},
		{
			handler: NewHandler(ctx2, mock, "", true),
			ct: testingcommon.Test{
				Name:        "Не успешно",
				Method:      http.MethodGet,
				Address:     address,
				ContentType: ContentTypeTextPlain,
				Want: testingcommon.Want{
					StatusCode: http.StatusInternalServerError,
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
			tt.handler.GetAllMetrics(w, r)

			result := w.Result()
			defer result.Body.Close()

			assert.Equal(t, tt.ct.Want.StatusCode, result.StatusCode)

		})
	}

}
