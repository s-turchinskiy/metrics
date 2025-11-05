// Package timeutils Функции для взаимодействия с датой-временем
package timeutils

import (
	"fmt"
	"time"
)

func ParseDurationFromString(durationStr string) (int, error) {
	if durationStr == "" {
		return 0, nil
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format '%s': %w", durationStr, err)
	}

	return int(duration.Seconds()), nil
}
