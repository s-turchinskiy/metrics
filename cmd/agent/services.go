package main

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"sync"
)

type MetricsUpdaterReporting interface {
	UpdateMetrics() error
	ReportMetrics() error
}
type MetricsStorage struct {
	Gauge         map[string]float64
	Counter       map[string]int64
	ServerAddress string
	mutex         sync.Mutex
}

func (s *MetricsStorage) ReportMetrics() error {

	client := resty.New()

	for name, value := range s.Gauge {

		url := path.Join(s.ServerAddress, "update", "gauge", name, strconv.FormatFloat(value, 'f', -1, 64))
		err := ReportMetric(client, url)
		if err != nil {
			return err
		}

	}

	for name, value := range s.Counter {

		url := path.Join(s.ServerAddress, "update", "gauge", name, strconv.FormatInt(value, 10))
		err := ReportMetric(client, url)
		if err != nil {
			return err
		}
	}

	return nil

}

func ReportMetric(client *resty.Client, url string) error {

	resp, err := client.R().
		SetHeader("Content-Type", "text/json").
		Post(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("status code <> 200, = %d, url : %s", resp.StatusCode(), url)
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
