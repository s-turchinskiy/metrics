package services

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/s-turchinskiy/metrics/cmd/agent/config"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/agent/retrier"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetric/httpresty"
	"github.com/s-turchinskiy/metrics/internal/agent/services/sendmetrics"
	"github.com/s-turchinskiy/metrics/internal/common"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"time"
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

type MetricsHandler struct {
	Storage       MetricsUpdaterReporting
	ServerAddress string
}

func ReportMetrics(h *MetricsHandler, errorsChan chan error) {

	ticker := time.NewTicker(time.Duration(config.ReportInterval) * time.Second)
	for range ticker.C {

		metrics, err := h.Storage.GetMetrics()
		if err != nil {
			logger.Log.Infoln("failed to report metrics", err.Error())
			errorsChan <- err
			return
		}

		sender := httpresty.New(
			fmt.Sprintf("%s/update/", h.ServerAddress),
			common.СomputeHexadecimalSha256Hash,
		)

		sendMetrics := sendmetrics.New(
			metrics,
			sender,
			retrier.ReportMetricRetry1{},
		)

		errs := sendMetrics.Send()
		sendMetrics.ErrorHandling(errs)

	}
}

func ReportMetricsBatch(h *MetricsHandler, errors chan error) {

	url := fmt.Sprintf("%s/updates/", h.ServerAddress)

	ticker := time.NewTicker(time.Duration(config.ReportInterval) * time.Second)
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

	ticker := time.NewTicker(time.Duration(config.PollInterval) * time.Second)
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
