package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port      int
	StreamURL string
}

type JSONConfig struct {
	Stream Stream `json:"stream"`
}

type Stream struct {
	URL string `json:"url"`
}

func LoadConfig() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}

	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		return Config{}, errors.New("CONFIG_FILE environment variable is required")
	}

	rawConfig, err := os.ReadFile(configFile)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var jsonConfig JSONConfig
	err = json.Unmarshal(rawConfig, &jsonConfig)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return Config{}, fmt.Errorf("invalid PORT value: %w", err)
	}

	return Config{
		Port:      portInt,
		StreamURL: jsonConfig.Stream.URL,
	}, nil
}
