package main

import (
	"fmt"
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

func ReportMetrics(h *MetricsHandler, mutex *sync.Mutex, errors chan error) {

	ticker := time.Tick(time.Duration(reportInterval) * time.Second)
	for range ticker {
		mutex.Lock()

		err := h.storage.ReportMetrics()
		if err != nil {
			errors <- err
			return
		}

		mutex.Unlock()

		fmt.Printf("\tReportMetrics, PollCount: %d\n", h.storage.(*MetricsStorage).Counter["PollCount"])
	}

}

func UpdateMetrics(h *MetricsHandler, mutex *sync.Mutex, errors chan error) {

	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	for range ticker.C {

		mutex.Lock()

		err := h.storage.UpdateMetrics()
		if err != nil {
			errors <- err
			return
		}

		mutex.Unlock()

		fmt.Printf("UpdateMetrics, PollCount: %d\n", h.storage.(*MetricsStorage).Counter["PollCount"])

	}

}
