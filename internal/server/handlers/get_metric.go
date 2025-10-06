package handlers

import (
	"net/http"
	"strings"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
)

// GetMetric godoc
// @Tags Info
// @Summary Получение значения метрики
// @Description Получение значения метрики по типу и наименованию метрики
// @ID infoGetMetric
// @Accept  json
// @Produce html
// @Param MetricsType path string true "Metrics Type" Enums(counter, gauge)
// @Param MetricsName path string true "Metrics Name"
// @Success 200 {string} string "100"
// @Failure 400 {string} string "Неверный запрос"
// @Failure 403 {string} string "Ошибка авторизации"
// @Failure 404 {string} string "Bucket не найден"
// @Failure 500 {string} string "Внутренняя ошибка"
// @Security ApiKeyAuth
// @Router /value/{MetricsType}/{MetricsName} [get]
func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", ContentTypeTextHTML)

	if r.Method != http.MethodGet {
		logger.Log.Infow("error, Method != Get", "Method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/value/")
	pathSlice := strings.Split(path, "/")

	metric := models.UntypedMetric{
		MetricsType: pathSlice[0],
		MetricsName: pathSlice[1],
	}

	value, err := h.Service.GetMetric(r.Context(), metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(value))

}
