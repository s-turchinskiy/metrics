package handlers

import (
	"encoding/json"
	"github.com/mailru/easyjson"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"go.uber.org/zap"
	"net/http"
)

func (h *MetricsHandler) GetTypedMetric(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		logger.Log.Infow("error, Method != Post", "Method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Info("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	metric := models.StorageMetrics{Name: req.ID, MType: req.MType, Delta: req.Delta, Value: req.Value}

	result, err := h.Service.GetTypedMetric(r.Context(), metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.Header().Set("Content-Type", ContentTypeTextHTML)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resp := &models.Metrics{ID: result.Name, MType: result.MType, Delta: result.Delta, Value: result.Value}
	rawBytes, err := easyjson.Marshal(resp)
	if err != nil {
		logger.Log.Info("error encoding response", zap.Error(err))
		return
	}

	w.Write(rawBytes)

}
