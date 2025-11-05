package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/common/errutil"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
)

// UpdateMetricJSON godoc
// @Tags Update
// @Summary Сохранение метрики
// @Description Создание новой / обновление существующей метрики из json
// @ID updateUpdateMetricJSON
// @Accept  json
// @Produce json
// @Param metric_data body models.Metrics true "Содержимое метрики"
// @Success 200 {object} models.Metrics "OK"
// @Failure 400 {string} string "Неверный запрос"
// @Failure 403 {string} string "Ошибка авторизации"
// @Failure 500 {string} string "Внутренняя ошибка"
// @Security ApiKeyAuth
// @Router /update [post]
func (h *MetricsHandler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {

	var req models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Log.Info("cannot decode request JSON body", zap.Error(errutil.WrapError(err)))
			logger.Log.Debugw(errutil.WrapError(fmt.Errorf("error read body")).Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		logger.Log.Info("cannot decode request JSON body", zap.Error(errutil.WrapError(err)), zap.String("body", string(body)))

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	metric := models.StorageMetrics{Name: req.ID, MType: req.MType, Delta: req.Delta, Value: req.Value}
	result, err := h.Service.UpdateTypedMetric(r.Context(), metric)
	if err != nil {
		logger.Log.Infoln("error", err.Error(), "metric", metric)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resp := models.Metrics{ID: result.Name, MType: result.MType, Delta: result.Delta, Value: result.Value}
	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		logger.Log.Info("error encoding response", zap.Error(err))
		return
	}

	if !settings.Settings.AsynchronousWritingDataToFile {
		err := h.Service.SaveMetricsToFile(r.Context())
		if err != nil {
			logger.Log.Info("error SaveMetricsToFile", zap.Error(err))
			return

		}
	}

}
