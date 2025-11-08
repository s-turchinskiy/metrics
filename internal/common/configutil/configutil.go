// Package configutil Функции для чтения в конфиг
package configutil

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func GetConfigFilePath() string {

	if value := os.Getenv("CONFIG"); value != "" {
		return value
	}

	var configJson string
	flag.StringVar(&configJson, "c", "", "Путь к json файлу с конфигурацией")
	flag.StringVar(&configJson, "configutil", "", "Путь к json файлу с конфигурацией")
	flag.Parse()

	return configJson
}

func LoadJSONConfig(filePath string, target any) error {

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open configutil file %s: %w", filePath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("cannot decode JSON configutil: %w", err)
	}

	return nil
}
