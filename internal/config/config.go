package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server struct {
		Port    string
		BaseURL string
	}
}

// retrieves an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func Load() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load()

	config := &Config{}

	// Server configuration
	config.Server.Port = getEnv("SERVER_PORT", ":8080")
	config.Server.BaseURL = getEnv("BASE_URL", "http://localhost:8080")

	// Audio configuration

	return config, nil
}
