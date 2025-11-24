// Package services Сервис обновления метрик
package services

import (
	"context"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/agent/reporter"
	"github.com/s-turchinskiy/metrics/internal/agent/repositories"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"reflect"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/s-turchinskiy/metrics/internal/agent/logger"
)

var (
	metricsNames = []string{"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse",
		"HeapObjects", "HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse", "MSpanSys",
		"Mallocs", "NextGC", "NumForcedGC", "NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys", "Sys", "TotalAlloc"}
)

type Service struct {
	storage       repositories.MetricsRepositorier
	reporter      reporter.Reporter
	serverAddress string
	pollInterval  int
}

func New(storage *repositories.MetricsStorage, reporter reporter.Reporter, addr string, pollInterval int) *Service {
	return &Service{
		storage:       storage,
		reporter:      reporter,
		serverAddress: addr,
		pollInterval:  pollInterval,
	}
}

func (s *Service) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.UpdateMetrics(ctx)
	})

	g.Go(func() error {
		return s.reporter.ReportMetrics(ctx)
	})

	g.Go(func() error {
		return s.reporter.ReportMetricsBatch(ctx)
	})

	return g.Wait()
}

// UpdateMetrics Обновление метрик в хранилище
func (s *Service) UpdateMetrics(ctx context.Context) error {

	ticker := time.NewTicker(time.Duration(s.pollInterval) * time.Second)
	for range ticker.C {

		select {
		case <-ctx.Done():
			return nil
		default:
			metrics, err := GetMetrics(1 * time.Second)
			if err != nil {
				logger.Log.Infoln("getMetrics error", err.Error())
			}

			err = s.storage.UpdateMetrics(metrics)
			if err != nil {
				logger.Log.Infoln("storage update metrics error", err.Error())
				return err
			}
		}

	}

	return nil
}

// GetMetrics Получение метрик из операционной системы
func GetMetrics(cpuTime time.Duration) (map[string]float64, error) {

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

	cpuPercent, err := cpu.Percent(cpuTime, true)
	if err != nil {
		return nil, err
	}
	for i, percent := range cpuPercent {
		result[fmt.Sprintf("CPUutilization%d", i)] = percent
	}

	return result, nil

}
