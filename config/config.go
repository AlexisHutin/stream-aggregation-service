package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port      int
	StreamURL string
}

func LoadConfig() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}

	streamURL := os.Getenv("STREAM_URL")
	if streamURL == "" {
		return Config{}, fmt.Errorf("STREAM_URL environment variable is required")
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return Config{}, fmt.Errorf("invalid PORT value: %v", err)
	}

	return Config{
		Port:      portInt,
		StreamURL: streamURL,
	}, nil
}
