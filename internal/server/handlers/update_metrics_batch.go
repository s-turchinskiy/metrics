package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
)

func (h *MetricsHandler) UpdateMetricsBatch(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		logger.Log.Infow("error, Method != Post", "Method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req []models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Info("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	metrics := make([]models.StorageMetrics, 0, len(req))
	for _, reqMetric := range req {
		metric := models.StorageMetrics{Name: reqMetric.ID, MType: reqMetric.MType, Delta: reqMetric.Delta, Value: reqMetric.Value}
		metrics = append(metrics, metric)
	}
	count, err := h.Service.UpdateTypedMetrics(r.Context(), metrics)
	if err != nil {
		logger.Log.Infoln("error", err.Error(), "metrics", metrics)
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("Load %d records", count)))

}
