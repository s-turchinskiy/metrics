package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"sync"
	"time"
)

type NetAddress struct {
	Host string
	Port int
}

var (
	pollInterval   int = 2
	reportInterval int = 10
)

var (
	metricsNames = []string{"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse",
		"HeapObjects", "HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse", "MSpanSys",
		"Mallocs", "NextGC", "NumForcedGC", "NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys", "Sys", "TotalAlloc"}
)

type MetricsHandler struct {
	storage MetricsUpdaterReporting
}

func main() {

	if err := logger.Initialize(); err != nil {
		panic(err)
	}

	addr := NetAddress{Host: "localhost", Port: 8080}
	parseFlags(&addr)

	var mutex sync.Mutex

	metricsHandler := &MetricsHandler{
		storage: &MetricsStorage{
			Gauge:         make(map[string]float64),
			Counter:       make(map[string]int64),
			ServerAddress: "http://" + addr.String(),
		},
	}

	errors := make(chan error)

	go UpdateMetrics(metricsHandler, &mutex, errors)
	go ReportMetrics(metricsHandler, &mutex, errors)

	err := <-errors
	panic(err)

}

func ReportMetric(client *resty.Client, ServerAddress string, metric models.Metrics) error {

	url := fmt.Sprintf("%s/update/", ServerAddress)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metric).
		Post(url)
	if err != nil {

		text := err.Error()
		var bytes []byte
		bytes, err := json.Marshal(metric)
		if err != nil {
			logger.Log.Infow("conversion error metric",
				"error", err.Error(),
				"url", url)
		}

		logger.Log.Infow("error sending request",
			"error", text,
			"url", url,
			"body", string(bytes))
		return err
	}

	if resp.StatusCode() != 200 {

		logger.Log.Infow("error. status code <> 200",
			"status code", resp.StatusCode(),
			"url", url,
			"body", string(resp.Body()))
		err := fmt.Errorf("status code <> 200, = %d, url : %s", resp.StatusCode(), url)
		return err
	}

	return nil

}

func ReportMetrics(h *MetricsHandler, mutex *sync.Mutex, errors chan error) {

	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	for range ticker.C {

		mutex.Lock()

		err := h.storage.ReportMetrics()
		if err != nil {
			logger.Log.Infoln("failed to report metrics", err.Error())
			errors <- err
			return
		}

		mutex.Unlock()
		logger.Log.Debugw("ReportMetrics", "PollCount", h.storage.(*MetricsStorage).Counter["PollCount"])

	}
}

func UpdateMetrics(h *MetricsHandler, mutex *sync.Mutex, errors chan error) {

	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	for range ticker.C {

		mutex.Lock()

		err := h.storage.UpdateMetrics()
		if err != nil {
			logger.Log.Infoln("error", err.Error())
			errors <- err
			return
		}

		mutex.Unlock()
		logger.Log.Debugw("UpdateMetrics", "PollCount", h.storage.(*MetricsStorage).Counter["PollCount"])

	}

}
