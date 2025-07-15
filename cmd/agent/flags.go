package main

import (
	"flag"
	"github.com/s-turchinskiy/metrics/internal/agent/services"
	"log"
	"os"
	"strconv"
)

func parseFlags(addr *services.NetAddress) {

	flag.Var(addr, "a", "Net address host:port")
	flag.IntVar(&services.PollInterval, "p", 2, "poll interval")
	flag.IntVar(&services.ReportInterval, "r", 10, "report interval")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		err := addr.Set(envAddr)
		if err != nil {
			log.Fatal(err)
		}
	}

	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		value, err := strconv.Atoi(envPollInterval)
		if err != nil {
			log.Fatal(err)
		}

		services.PollInterval = value
	}

	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		value, err := strconv.Atoi(envReportInterval)
		if err != nil {
			log.Fatal(err)
		}

		services.ReportInterval = value
	}

}
