package util

import (
	"fmt"
	"time"
)

// ParseDuration parses the given option name and value
func ParseDuration(optionName string, optionValue string) (*time.Duration, error) {
	if optionValue == "" {
		return nil, nil
	}
	duration, err := time.ParseDuration(optionValue)
	if err != nil {
		return nil, fmt.Errorf("Invalid duration format %s for option --%s: %s", optionValue, optionName, err)
	}
	return &duration, nil
}
