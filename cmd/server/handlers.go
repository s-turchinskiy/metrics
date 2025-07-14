package main

import (
	"encoding/json"
	"fmt"
	"github.com/mailru/easyjson"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
)

func (h *MetricsHandler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html")

	if r.URL.Path != "/" {
		logger.Log.Infow("error, Path != \"/\"", "path", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		logger.Log.Infow("error, Method != Get", "Method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	result := h.storage.GetAllMetrics()
	w.WriteHeader(http.StatusOK)

	for mtype, table := range result {
		io.WriteString(w, fmt.Sprintf("<div>%s</div>", mtype))
		io.WriteString(w, "<table style=\"margin-left: 40px\">")
		for name, value := range table {
			io.WriteString(w, fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", name, value))
		}
		io.WriteString(w, "</table>")
	}

	/*for _, str := range metrics {
		runes := []rune(str)
		if runes[0] == '\t' {
			io.WriteString(w, fmt.Sprintf("%s", str))
		} else {

		}
	}*/

}

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		logger.Log.Infow("error, Method != Post", "Method", r.Method)
		w.Header().Set("Content-Type", "text/html")
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

	err := h.storage.UpdateMetric(metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

}

func (h *MetricsHandler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		logger.Log.Infow("error, Method != Post", "Method", r.Method)
		w.Header().Set("Content-Type", "text/html")
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
	result, err := h.storage.UpdateTypedMetric(metric)
	if err != nil {
		logger.Log.Infoln("error", err.Error())
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := models.Metrics{ID: result.Name, MType: result.MType, Delta: result.Delta, Value: result.Value}
	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		logger.Log.Info("error encoding response", zap.Error(err))
		return
	}

	if !settings.asynchronousWritingDataToFile {
		err := h.storage.SaveMetricsToFile()
		if err != nil {
			logger.Log.Info("error SaveMetricsToFile", zap.Error(err))
			return

		}
	}

}

func (h *MetricsHandler) GetTypedMetric(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		logger.Log.Infow("error, Method != Post", "Method", r.Method)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Info("cannot decode request JSON body", zap.Error(err))
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	metric := models.StorageMetrics{Name: req.ID, MType: req.MType, Delta: req.Delta, Value: req.Value}

	result, err := h.storage.GetTypedMetric(metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.Header().Set("Content-Type", "text/html")
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

	w.WriteHeader(http.StatusOK)
	w.Write(rawBytes)

}

func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html")

	if r.Method != http.MethodGet {
		logger.Log.Infow("error, Method != Get", "Method", r.Method)
		w.Header().Set("Content-Type", "text/html")
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

	value, err := h.storage.GetMetric(metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(value))

}

func (h *MetricsHandler) Ping(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Method != http.MethodGet {
		logger.Log.Infow("error, Method != Get", "Method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := h.db.Ping()

	if err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("<div>" + err.Error() + "</div>"))
		return
	}

	stats, err := json.MarshalIndent(h.db.Stats(), "", "   ")
	if err == nil {
		result := []byte("<div>")
		result = append(result, stats...)
		result = append(result, []byte("</div>")...)
		w.Write(result)
	}
}
