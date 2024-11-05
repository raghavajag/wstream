package config

import (
	"fmt"
	"os"
	"strconv"
)

type AudioConfig struct {
	BufferSize int
	FFmpegPath string
}

type Config struct {
	Port  int
	Audio AudioConfig
}

func Load() (*Config, error) {
	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT configuration: %v", err)
	}

	return &Config{
		Port: port,
		Audio: AudioConfig{
			BufferSize: getEnvAsInt("BUFFER_SIZE", 1024*1024), // 1MB
			FFmpegPath: getEnv("FFMPEG_PATH", "/usr/bin/ffmpeg"),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
