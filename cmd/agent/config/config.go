// Package config Конфигурирование текущего сервиса
package config

import (
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/utils/configutil"
	"github.com/s-turchinskiy/metrics/internal/utils/rsautil"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type NetAddress struct {
	Host string
	Port int
}

type ProgramConfig struct {
	Addr             *NetAddress
	PollInterval     int
	ReportInterval   int
	HashKey          string
	RateLimit        int //Количество одновременно исходящих запросов на сервер
	rsaPublicKeyPath string
	RSAPublicKey     *rsa.PublicKey
}

func ParseFlags() (*ProgramConfig, error) {

	cfg := ProgramConfig{}

	cfg.Addr = &NetAddress{Host: "localhost", Port: 8080}

	configFilePath := configutil.GetConfigFilePath()
	if configFilePath != "" {
		if err := loadConfigFromJSON(&cfg, configFilePath); err != nil {
			return nil, fmt.Errorf("failed to load configutil from JSON: %w", err)
		}
	}

	flag.Var(cfg.Addr, "a", "Net address host:port")
	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval")
	flag.StringVar(&cfg.HashKey, "k", "", "HashSHA256 key")
	flag.IntVar(&cfg.RateLimit, "l", runtime.NumCPU(), "number of concurrently outgoing requests to server")
	flag.StringVar(&cfg.rsaPublicKeyPath, "crypto-key", "", "Путь до файла с публичным ключом")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		err := cfg.Addr.Set(envAddr)
		if err != nil {
			return nil, err
		}
	}

	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		value, err := strconv.Atoi(envPollInterval)
		if err != nil {
			return nil, err
		}

		cfg.PollInterval = value
	}

	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		value, err := strconv.Atoi(envReportInterval)
		if err != nil {
			return nil, err
		}

		cfg.ReportInterval = value
	}

	if valueStr := os.Getenv("RATE_LIMIT"); valueStr != "" {
		value, err := strconv.Atoi(valueStr)
		if err != nil {
			return nil, err
		}

		cfg.RateLimit = value
	}

	if value := os.Getenv("KEY"); value != "" {
		cfg.HashKey = value
	}

	if value := os.Getenv("CRYPTO_KEY"); value != "" {
		cfg.rsaPublicKeyPath = value
	}

	if cfg.rsaPublicKeyPath != "" {
		var err error
		cfg.RSAPublicKey, err = rsautil.ReadPublicKey(cfg.rsaPublicKeyPath)
		if err != nil {
			err = fmt.Errorf("path: %s, error: %w", cfg.rsaPublicKeyPath, err)
			return nil, err
		}
	}

	return &cfg, nil
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
