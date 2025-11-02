package config

import (
	configutils "github.com/s-turchinskiy/metrics/internal/common/config"
	timeutils "github.com/s-turchinskiy/metrics/internal/common/time"
)

type JSONConfig struct {
	Address        string `json:"address,omitempty"`
	ReportInterval string `json:"report_interval,omitempty"`
	PollInterval   string `json:"poll_interval,omitempty"`
	CryptoKey      string `json:"crypto_key,omitempty"`
}

func loadConfigFromJSON(config *ProgramConfig, filePath string) error {
	var jsonConfig JSONConfig
	if err := configutils.LoadJSONConfig(filePath, &jsonConfig); err != nil {
		return err
	}

	if jsonConfig.Address != "" {
		err := config.Addr.Set(jsonConfig.Address)
		if err != nil {
			return err
		}
	}

	if jsonConfig.CryptoKey != "" {
		config.rsaPublicKeyPath = jsonConfig.CryptoKey
	}

	if jsonConfig.ReportInterval != "" {
		seconds, err := timeutils.ParseDurationFromString(jsonConfig.ReportInterval)
		if err != nil {
			return err
		}
		config.ReportInterval = seconds
	}

	if jsonConfig.PollInterval != "" {
		seconds, err := timeutils.ParseDurationFromString(jsonConfig.PollInterval)
		if err != nil {
			return err
		}
		config.PollInterval = seconds
	}

	return nil
}
