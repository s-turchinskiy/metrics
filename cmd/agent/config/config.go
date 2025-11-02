// Package config Конфигурирование текущего сервиса
package config

import (
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	rsautil "github.com/s-turchinskiy/metrics/internal/common/rsa"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type NetAddress struct {
	Host string
	Port int
}

var (
	PollInterval     int = 2
	ReportInterval   int = 10
	HashKey          string
	RateLimit        int //Количество одновременно исходящих запросов на сервер
	rsaPublicKeyPath string
	RSAPublicKey     *rsa.PublicKey
)

func ParseFlags(addr *NetAddress) {

	flag.Var(addr, "a", "Net address host:port")
	flag.IntVar(&PollInterval, "p", 2, "poll interval")
	flag.IntVar(&ReportInterval, "r", 10, "report interval")
	flag.StringVar(&HashKey, "k", "", "HashSHA256 key")
	flag.IntVar(&RateLimit, "l", runtime.NumCPU(), "number of concurrently outgoing requests to server")
	flag.StringVar(&rsaPublicKeyPath, "crypto-key", "", "Путь до файла с публичным ключом")
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

	if valueStr := os.Getenv("RATE_LIMIT"); valueStr != "" {
		value, err := strconv.Atoi(valueStr)
		if err != nil {
			log.Fatal(err)
		}

		RateLimit = value
	}

	if value := os.Getenv("KEY"); value != "" {
		HashKey = value
	}

	if value := os.Getenv("CRYPTO_KEY"); value != "" {
		rsaPublicKeyPath = value
	}

	if rsaPublicKeyPath != "" {
		var err error
		RSAPublicKey, err = rsautil.ReadPublicKey(rsaPublicKeyPath)
		if err != nil {
			err = fmt.Errorf("path: %s, error: %w", rsaPublicKeyPath, err)
			log.Fatal(err)
		}
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
