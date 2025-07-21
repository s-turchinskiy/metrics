package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	PollInterval   int = 2
	ReportInterval int = 10
)

var (
	metricsNames = []string{"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse",
		"HeapObjects", "HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse", "MSpanSys",
		"Mallocs", "NextGC", "NumForcedGC", "NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys", "Sys", "TotalAlloc"}
)

type MetricsUpdaterReporting interface {
	UpdateMetrics(map[string]float64) error
	GetMetrics() ([]models.Metrics, error)
}

type NetAddress struct {
	Host string
	Port int
}

func (a *NetAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *NetAddress) Set(s string) error {
	hp := strings.Split(s, ":")
	if len(hp) != 2 {
		return errors.New("need address in a form host:port")
	}
	port, err := strconv.Atoi(hp[1])
	if err != nil {
		return err
	}
	a.Host = hp[0]
	a.Port = port
	return nil
}

type MetricsHandler struct {
	Storage       MetricsUpdaterReporting
	ServerAddress string
}

func reportMetric(client *resty.Client, ServerAddress string, metric models.Metrics) error {

	url := fmt.Sprintf("%s/update/", ServerAddress)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metric).
		Post(url)

	if err != nil {

		text := err.Error()
		var bytes []byte
		bytes, err2 := json.Marshal(metric)
		if err2 != nil {
			logger.Log.Infow("conversion error metric",
				"error", err2.Error(),
				"url", url,
			)
		}

		logger.Log.Infow("error sending request",
			"error", text,
			"url", url,
			"body", string(bytes))
		return err
	}

	if resp.StatusCode() != http.StatusOK {

		logger.Log.Infow("error. status code <> 200",
			"status code", resp.StatusCode(),
			"url", url,
			"body", string(resp.Body()))
		err := fmt.Errorf("status code <> 200, = %d, url : %s", resp.StatusCode(), url)
		return err
	}

	return nil

}

func ReportMetrics(h *MetricsHandler, errorsChan chan error) {

	ticker := time.NewTicker(time.Duration(ReportInterval) * time.Second)
	for range ticker.C {

		client := resty.New()

		metrics, err := h.Storage.GetMetrics()
		if err != nil {
			logger.Log.Infoln("failed to report metrics", err.Error())
			errorsChan <- err
			return
		}

		result := make(chan error, len(metrics))
		wg := sync.WaitGroup{}
		wg.Add(len(metrics))

		for _, metric := range metrics {
			go func() {
				defer wg.Done()
				result <- reportMetricSeveralAttempts(client, h.ServerAddress, metric)
			}()
		}

		wg.Wait()
		close(result)

		var errs []error
		for err = range result {
			if err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) != 0 {
			logger.Log.Info("errors sending data ", errors.Join(errs...))
			//errorsChan <- errors.Join(errs...)
			return
		}

		logger.Log.Info("Success ReportMetrics")

	}
}

func reportMetricSeveralAttempts(client *resty.Client, serverAddress string, metric models.Metrics) error {

	logger.Log.Debugw("reportMetric. attempt 1", "data", metric)

	err := reportMetric(client, serverAddress, metric)
	if itIsErrorConnectionRefused(err) {
		logger.Log.Infow("reportMetric, server is not responding. attempt 2", "data", metric)
		time.Sleep(1 * time.Second)
		err = reportMetric(client, serverAddress, metric)
		if itIsErrorConnectionRefused(err) {
			logger.Log.Infow("reportMetric, server is not responding. attempt 3", "data", metric)
			time.Sleep(3 * time.Second)
			err = reportMetric(client, serverAddress, metric)
			if itIsErrorConnectionRefused(err) {
				logger.Log.Infow("reportMetric, server is not responding. attempt 4", "data", metric)
				time.Sleep(5 * time.Second)
				err = reportMetric(client, serverAddress, metric)
			}
		}
	}

	return err

}

func itIsErrorConnectionRefused(err error) bool {

	return err != nil &&
	(strings.Contains(err.Error(), "connect: connection refused")) || strings.Contains(err.Error(), "connection reset by peer"))
}

func ReportMetricsBatch(h *MetricsHandler, errors chan error) {

	url := fmt.Sprintf("%s/updates/", h.ServerAddress)

	ticker := time.NewTicker(time.Duration(ReportInterval) * time.Second)
	for range ticker.C {

		client := resty.New()

		metrics, err := h.Storage.GetMetrics()
		if err != nil {
			logger.Log.Infoln("failed to report metrics batch", err.Error())
			errors <- err
			return
		}

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(metrics).
			Post(url)

		if err != nil {

			var bytes []byte
			bytes, err2 := json.Marshal(metrics)
			if err2 != nil {
				logger.Log.Infow("conversion error metric",
					"error", err2.Error(),
					"url", url,
				)
			}

			logger.Log.Infow("error sending request",
				"error", err.Error(),
				"url", url,
				"body", string(bytes))

			errors <- err
			return
		}

		if resp.StatusCode() != http.StatusOK {

			logger.Log.Infow("error. status code <> 200",
				"status code", resp.StatusCode(),
				"url", url,
				"body", string(resp.Body()))
			err := fmt.Errorf("status code <> 200, = %d, url : %s", resp.StatusCode(), url)

			errors <- err
			return

		}

		logger.Log.Info("Success ReportMetricsBatch ", string(resp.Body()))
	}
}

func UpdateMetrics(h *MetricsHandler, errors chan error) {

	ticker := time.NewTicker(time.Duration(PollInterval) * time.Second)
	for range ticker.C {

		metrics, err := GetMetrics()
		if err != nil {
			logger.Log.Infoln("getMetrics error", err.Error())
		}

		err = h.Storage.UpdateMetrics(metrics)
		if err != nil {
			logger.Log.Infoln("storage update metrics error", err.Error())
			errors <- err
			return
		}

	}

}

func GetMetrics() (map[string]float64, error) {

	result := make(map[string]float64, len(metricsNames))

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
					result[metricsName] = float64(v.Field(i).Interface().(uint64))
				}
			case "uint32":
				{
					result[metricsName] = float64(v.Field(i).Interface().(uint32))
				}
			case "float64":
				{
					result[metricsName] = v.Field(i).Interface().(float64)
				}

			default:
				return nil, fmt.Errorf("unexpected type %s for metric %s", typeName, metricsName)
			}

		}
	}

	result["RandomValue"] = rand.Float64()

	return result, nil

}
