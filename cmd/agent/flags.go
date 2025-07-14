package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
)

func parseFlags(addr *NetAddress) {

	flag.Var(addr, "a", "Net address host:port")
	flag.IntVar(&pollInterval, "p", 2, "poll interval")
	flag.IntVar(&reportInterval, "r", 10, "report interval")
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

		pollInterval = value
	}

	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		value, err := strconv.Atoi(envReportInterval)
		if err != nil {
			log.Fatal(err)
		}

		reportInterval = value
	}

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
