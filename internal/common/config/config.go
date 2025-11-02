package configutils

import (
	"encoding/json"
	"fmt"
	"os"
)

func GetConfigFilePath() string {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" || args[i] == "-config" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
	}

	return os.Getenv("CONFIG")
}

func LoadJSONConfig(filePath string, target any) error {

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open config file %s: %w", filePath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("cannot decode JSON config: %w", err)
	}

	return nil
}
