package settings

import (
	configutils "github.com/s-turchinskiy/metrics/internal/common/configutil"
	timeutils "github.com/s-turchinskiy/metrics/internal/common/timeutil"
)

type JSONConfig struct {
	Address       string `json:"address,omitempty"`
	Restore       bool   `json:"restore,omitempty"`
	StoreInterval string `json:"store_interval,omitempty"`
	StoreFile     string `json:"store_file,omitempty"`
	DatabaseDSN   string `json:"database_dsn,omitempty"`
	CryptoKey     string `json:"crypto_key,omitempty"`
}

func loadConfigFromJSON(config *ProgramSettings, filePath string) error {
	var jsonConfig JSONConfig
	if err := configutils.LoadJSONConfig(filePath, &jsonConfig); err != nil {
		return err
	}

	if jsonConfig.Address != "" {
		err := config.Address.Set(jsonConfig.Address)
		if err != nil {
			return err
		}
	}

	if jsonConfig.Restore {
		config.Restore = jsonConfig.Restore
	}

	if jsonConfig.StoreFile != "" {
		config.FileStoragePath = jsonConfig.StoreFile
	}

	if jsonConfig.DatabaseDSN != "" {
		err := config.Database.Set(jsonConfig.DatabaseDSN)
		if err != nil {
			return err
		}
	}

	if jsonConfig.CryptoKey != "" {
		config.RSAPrivateKeyPath = jsonConfig.CryptoKey
	}

	if jsonConfig.StoreInterval != "" {
		seconds, err := timeutils.ParseDurationFromString(jsonConfig.StoreInterval)
		if err != nil {
			return err
		}
		config.StoreInterval = seconds
	}

	return nil
}
