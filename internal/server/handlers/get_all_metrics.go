package handlers

import (
	"bytes"
	"html/template"
	"net/http"

	"go.uber.org/zap"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
)

type OutputAllMetrics struct {
	Header string
	Table  map[string]string
}

//type output map[string]map[string]float64`example:"key:value,key2:value2"`

// GetAllMetrics godoc
// @Tags Info
// @Summary Получение всех метрик на текущий момент
// @ID infoGetAllMetrics
// @Accept  json
// @Produce html
// @Success 200 {object} map[string]map[string]float64 "Counter PollCount 3119064 someMetric	26 Gauge Alloc	2829408 BuckHashSys	3349 CPUutilization0	9.708737864123963"
// @Failure 500 {string} string "Внутренняя ошибка"
// @Router / [get]
func (h *MetricsHandler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html")

	if r.URL.Path != "/" {
		logger.Log.Infow("error, Path != \"/\"", "path", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	result, err := h.Service.GetAllMetrics(r.Context())
	if err != nil {
		logger.Log.Info("error getting data", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for mtype, table := range result {

		var body bytes.Buffer

		data := OutputAllMetrics{Header: mtype, Table: table}

		t := template.Must(template.New("").Parse(templateOutputAllMetrics))
		if err := t.Execute(&body, data); err != nil {
			logger.Log.Info("cannot output data", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.Write(body.Bytes())
	}

}
