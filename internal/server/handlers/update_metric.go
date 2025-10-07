package handlers

import (
	"net/http"
	"strings"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
)

// UpdateMetric godoc
// @Tags Update
// @Summary Обновление значения метрики
// @Description Обновление значения метрики с передачей в параметрах запроса типа, наименования, значения метрики
// @ID updateUpdateMetric
// @Accept  json
// @Produce html
// @Param MetricsType path string true "Metrics Type" Enums(counter, gauge)
// @Param MetricsName path string true "Metrics Name"
// @Param MetricsValue path float64 true "Metrics Value"
// @Success 200 {Object} string "100, установленное значение метрики"
// @Failure 400 {string} string "Неверный запрос"
// @Failure 403 {string} string "Ошибка авторизации"
// @Failure 404 {string} string "Bucket не найден"
// @Failure 500 {string} string "Внутренняя ошибка"
// @Security ApiKeyAuth
// @Router /value/{MetricsType}/{MetricsName}/{MetricsValue} [get]
func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", ContentTypeTextHTML)

	if r.Method != http.MethodPost {
		logger.Log.Infow("error, Method != Post", "Method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/update/")
	pathSlice := strings.Split(path, "/")

	metric := models.UntypedMetric{
		MetricsType:  pathSlice[0],
		MetricsName:  pathSlice[1],
		MetricsValue: pathSlice[2], //r.PathValue("MetricsValue"),
	}

	err := h.Service.UpdateMetric(r.Context(), metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

}
