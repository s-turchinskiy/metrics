package handlers

import (
	"net/http"
	"strings"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
)

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
