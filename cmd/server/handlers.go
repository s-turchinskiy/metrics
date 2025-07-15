package main

import (
	"bytes"
	"encoding/json"
	"github.com/mailru/easyjson"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"go.uber.org/zap"
	"html/template"
	"net/http"
	"strings"
)

type OutputAllMetrics struct {
	Header string
	Table  map[string]string
}

const contentTypeTextHTML = "text/html; charset=utf-8"

var (
	templateOutputAllMetrics = `<div>{{.Header}}</div><table style="margin-left: 40px">{{range $k, $v:= .Table}}<tr><td>{{$k}}</td><td>{{$v}}</td></tr>{{end}}</table>`
)

func (h *MetricsHandler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", contentTypeTextHTML)

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

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", contentTypeTextHTML)

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

	err := h.storage.UpdateMetric(metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

}

func (h *MetricsHandler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {

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

	result, err := h.storage.GetTypedMetric(metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.Header().Set("Content-Type", contentTypeTextHTML)
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

func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", contentTypeTextHTML)

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

	value, err := h.storage.GetMetric(metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(value))

}

func (h *MetricsHandler) Ping(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", contentTypeTextHTML)

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
