package services

import (
	"fmt"
	"github.com/s-turchinskiy/metrics/cmd/agent/config"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"math/rand"
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

func UpdateMetrics(h *MetricsHandler, errors chan error, doneCh chan struct{}) {

	ticker := time.NewTicker(time.Duration(config.PollInterval) * time.Second)
	for range ticker.C {

		select {
		case <-doneCh:
			return
		default:
			metrics, err := GetMetrics()
			if err != nil {
				logger.Log.Infoln("getMetrics error", err.Error())
			}

			err = h.Storage.UpdateMetrics(metrics)
			if err != nil {
				logger.Log.Infoln("storage update metrics error", err.Error())
				doneCh <- struct{}{}
				errors <- err
				return
			}
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

	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	result["TotalMemory"] = float64(vm.Total)
	result["FreeMemory"] = float64(vm.Free)

	cpuPercent, err := cpu.Percent(1*time.Second, true)
	if err != nil {
		return nil, err
	}
	for i, percent := range cpuPercent {
		result[fmt.Sprintf("CPUutilization%d", i)] = percent
	}

	return result, nil

}
