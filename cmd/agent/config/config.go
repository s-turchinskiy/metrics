package config

import (
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
)

type NetAddress struct {
	Host string
	Port int
}

var (
	PollInterval   int = 2
	ReportInterval int = 10
	HashKey        string
)

func ParseFlags(addr *NetAddress) {

	flag.Var(addr, "a", "Net address host:port")
	flag.IntVar(&PollInterval, "p", 2, "poll interval")
	flag.IntVar(&ReportInterval, "r", 10, "report interval")
	flag.StringVar(&HashKey, "k", "", "HashSHA256 key")
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

		PollInterval = value
	}

	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		value, err := strconv.Atoi(envReportInterval)
		if err != nil {
			log.Fatal(err)
		}

		ReportInterval = value
	}

	if value := os.Getenv("KEY"); value != "" {
		HashKey = value
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
