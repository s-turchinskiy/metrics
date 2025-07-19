package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/mailru/easyjson"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"github.com/s-turchinskiy/metrics/internal/server/repository/memcashed"
	"github.com/s-turchinskiy/metrics/internal/server/repository/postgresql"
	"github.com/s-turchinskiy/metrics/internal/server/service"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"go.uber.org/zap"
	"html/template"
	"log"
	"net/http"
	"strings"
)

type OutputAllMetrics struct {
	Header string
	Table  map[string]string
}

type MetricsHandler struct {
	Service service.MetricsUpdater
}

const ContentTypeTextHTML = "text/html; charset=utf-8"

var (
	templateOutputAllMetrics = `<div>{{.Header}}</div><table style="margin-left: 40px">{{range $k, $v:= .Table}}<tr><td>{{$k}}</td><td>{{$v}}</td></tr>{{end}}</table>`
)

func NewHandler(ctx context.Context) *MetricsHandler {
	metricsHandler := &MetricsHandler{}
	if settings.Settings.Store == settings.Database {

		p, err := postgresql.Initialize(ctx)
		if err != nil {
			logger.Log.Debugw("Connect to database error", "error", err.Error())
			log.Fatal(err)
		}

		metricsHandler.Service = &service.Service{
			Repository: p,
		}

	} else {

		metricsHandler.Service = &service.Service{
			Repository: &memcashed.MemCashed{
				Gauge:   make(map[string]float64),
				Counter: make(map[string]int64),
			},
		}

		if settings.Settings.Restore {
			err := metricsHandler.Service.LoadMetricsFromFile(ctx)
			if err != nil {
				logger.Log.Errorw("LoadMetricsFromFile error", "error", err.Error())
				log.Fatal(err)
			}
		}
	}

	return metricsHandler

}

func (h *MetricsHandler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

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
	result, err := h.Service.UpdateTypedMetric(r.Context(), metric)
	if err != nil {
		logger.Log.Infoln("error", err.Error(), "metric", metric)
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

	if !settings.Settings.AsynchronousWritingDataToFile {
		err := h.Service.SaveMetricsToFile(r.Context())
		if err != nil {
			logger.Log.Info("error SaveMetricsToFile", zap.Error(err))
			return

		}
	}

}

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
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Load %d records", count)))

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

	result, err := h.Service.GetTypedMetric(r.Context(), metric)
	if err != nil {
		logger.Log.Infoln(err.Error())
		w.Header().Set("Content-Type", ContentTypeTextHTML)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := &models.Metrics{ID: result.Name, MType: result.MType, Delta: result.Delta, Value: result.Value}
	rawBytes, err := easyjson.Marshal(resp)
	if err != nil {
		logger.Log.Info("error encoding response", zap.Error(err))
		return
	}

	w.Write(rawBytes)

}

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

func (h *MetricsHandler) Ping(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", ContentTypeTextHTML)

	if r.Method != http.MethodGet {
		logger.Log.Infow("error, Method != Get", "Method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	data, err := h.Service.Ping(r.Context())

	if err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("<div>" + err.Error() + "</div>"))
		return
	}

	result := []byte("<div>")
	result = append(result, data...)
	result = append(result, []byte("</div>")...)
	w.Write(result)

}
