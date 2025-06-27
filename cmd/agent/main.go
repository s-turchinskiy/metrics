package main

import (
	"fmt"
	"github.com/go-resty/resty/v2"
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
				errors <- err
				return
			}

			mutex.Unlock()

			time.Sleep(time.Duration(pollInterval) * time.Second)
			fmt.Printf("UpdateMetrics, PollCount: %d\n", metricsHandler.storage.(*MetricsStorage).Counter["PollCount"])

		}

	}()

	go func() {

		for {

			mutex.Lock()

			err := metricsHandler.storage.ReportMetrics()
			if err != nil {
				errors <- err
				return
			}

			mutex.Unlock()

			time.Sleep(time.Duration(reportInterval) * time.Second)
			fmt.Printf("\tReportMetrics, PollCount: %d\n", metricsHandler.storage.(*MetricsStorage).Counter["PollCount"])

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

func (s *MetricsStorage) ReportMetrics() error {

	client := resty.New()

	for name, value := range s.Gauge {

		url := fmt.Sprintf("%s/update/%s/%s/%f", s.ServerAddress, "gauge", name, value)
		resp, err := client.R().
			SetHeader("Content-Type", "text/json").
			Post(url)
		if err != nil {
			return err
		}

		if resp.StatusCode() != 200 {
			return fmt.Errorf("status code <> 200, = %d, url : %s", resp.StatusCode(), url)
		}
	}

	for name, value := range s.Counter {

		url := fmt.Sprintf("%s/update/%s/%s/%d", s.ServerAddress, "counter", name, value)
		resp, err := client.R().
			SetHeader("Content-Type", "text/json").
			Post(url)
		if err != nil {
			return err
		}

		if resp.StatusCode() != 200 {
			return fmt.Errorf("status code <> 200, = %d, url : %s", resp.StatusCode(), url)
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
