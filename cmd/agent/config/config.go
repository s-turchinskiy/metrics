// Package config Конфигурирование текущего сервиса
package config

import (
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	configutils "github.com/s-turchinskiy/metrics/internal/common/config"
	rsautil "github.com/s-turchinskiy/metrics/internal/common/rsa"
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

var Config ProgramConfig

func ParseFlags() error {

	Config.Addr = &NetAddress{Host: "localhost", Port: 8080}

	configFilePath := configutils.GetConfigFilePath()
	if configFilePath != "" {
		if err := loadConfigFromJSON(&Config, configFilePath); err != nil {
			return fmt.Errorf("failed to load config from JSON: %w", err)
		}
	}

	flag.Var(Config.Addr, "a", "Net address host:port")
	flag.IntVar(&Config.PollInterval, "p", 2, "poll interval")
	flag.IntVar(&Config.ReportInterval, "r", 10, "report interval")
	flag.StringVar(&Config.HashKey, "k", "", "HashSHA256 key")
	flag.IntVar(&Config.RateLimit, "l", runtime.NumCPU(), "number of concurrently outgoing requests to server")
	flag.StringVar(&Config.rsaPublicKeyPath, "crypto-key", "", "Путь до файла с публичным ключом")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		err := Config.Addr.Set(envAddr)
		if err != nil {
			return err
		}
	}

	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		value, err := strconv.Atoi(envPollInterval)
		if err != nil {
			return err
		}

		Config.PollInterval = value
	}

	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		value, err := strconv.Atoi(envReportInterval)
		if err != nil {
			return err
		}

		Config.ReportInterval = value
	}

	if valueStr := os.Getenv("RATE_LIMIT"); valueStr != "" {
		value, err := strconv.Atoi(valueStr)
		if err != nil {
			return err
		}

		Config.RateLimit = value
	}

	if value := os.Getenv("KEY"); value != "" {
		Config.HashKey = value
	}

	if value := os.Getenv("CRYPTO_KEY"); value != "" {
		Config.rsaPublicKeyPath = value
	}

	if Config.rsaPublicKeyPath != "" {
		var err error
		Config.RSAPublicKey, err = rsautil.ReadPublicKey(Config.rsaPublicKeyPath)
		if err != nil {
			err = fmt.Errorf("path: %s, error: %w", Config.rsaPublicKeyPath, err)
			return err
		}
	}

	return nil
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
