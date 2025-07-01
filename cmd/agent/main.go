package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/s-turchinskiy/metrics/cmd/agent/internal/logger"
	"github.com/s-turchinskiy/metrics/models"
	"reflect"
	"runtime"
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

type MetricsUpdaterReporting interface {
	UpdateMetrics() error
	ReportMetrics() error
}

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

	go func() {

		for {

			mutex.Lock()

			err := metricsHandler.storage.UpdateMetrics()
			if err != nil {
				logger.Log.Error("error", err.Error())
				errors <- err
				return
			}

			mutex.Unlock()

			time.Sleep(time.Duration(pollInterval) * time.Second)
			logger.Log.Infow("UpdateMetrics", "PollCount", metricsHandler.storage.(*MetricsStorage).Counter["PollCount"])

		}

	}()

	go func() {

		for {

			mutex.Lock()

			err := metricsHandler.storage.ReportMetrics()
			if err != nil {
				logger.Log.Error("error", err.Error())
				errors <- err
				return
			}

			mutex.Unlock()

			time.Sleep(time.Duration(reportInterval) * time.Second)
			logger.Log.Infow("ReportMetrics", "PollCount", metricsHandler.storage.(*MetricsStorage).Counter["PollCount"])

		}
	}()

	err := <-errors
	panic(err)

}

type MetricsStorage struct {
	Gauge         map[string]float64
	Counter       map[string]int64
	ServerAddress string
}

func ReportMetric(client *resty.Client, ServerAddress string, metric models.Metrics) error {

	url := fmt.Sprintf("%s/update/", ServerAddress)
	bytes, _ := json.Marshal(metric)
	logger.Log.Infoln(
		"url", url,
		"method", "Post",
		"body", string(bytes),
	)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metric).
		Post(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {

		return fmt.Errorf("status code <> 200, = %d, url : %s", resp.StatusCode(), url)
	}

	return nil

}

func (s *MetricsStorage) ReportMetrics() error {

	client := resty.New()

	for ID, value := range s.Gauge {

		metric := models.Metrics{ID: ID, MType: "gauge", Value: &value}
		err := ReportMetric(client, s.ServerAddress, metric)
		if err != nil {
			return err
		}
	}

	for ID, value := range s.Counter {

		metric := models.Metrics{ID: ID, MType: "counter", Delta: &value}
		err := ReportMetric(client, s.ServerAddress, metric)
		if err != nil {
			return err
		}
	}

	return nil

}

func (s *MetricsStorage) UpdateMetrics() error {

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	v := reflect.ValueOf(memStats)

	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		for _, metricsName := range metricsNames {
			if metricsName != typeOfS.Field(i).Name {
				continue
			}

			switch typeName := typeOfS.Field(i).Type.Name(); typeName {

			case "uint64":
				{
					s.Gauge[metricsName] = float64(v.Field(i).Interface().(uint64))
				}
			case "uint32":
				{
					s.Gauge[metricsName] = float64(v.Field(i).Interface().(uint32))
				}
			case "float64":
				{
					s.Gauge[metricsName] = v.Field(i).Interface().(float64)
				}

			default:
				return fmt.Errorf("unexpected type %s for metric %s", typeName, metricsName)
			}

		}
	}

	s.Counter["PollCount"]++

	return nil
}
