package main

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"math/rand"
	"reflect"
	"runtime"
)

type MetricsUpdaterReporting interface {
	UpdateMetrics() error
	ReportMetrics() error
}

type MetricsStorage struct {
	Gauge         map[string]float64
	Counter       map[string]int64
	ServerAddress string
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

	s.Gauge["RandomValue"] = rand.Float64()
	s.Counter["PollCount"]++

	return nil
}
