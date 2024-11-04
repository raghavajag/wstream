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
	Audio struct {
		BufferSize    int
		Channels      int
		SampleRate    int
		BitsPerSample int
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
	config.Audio.BufferSize = getEnvAsInt("AUDIO_BUFFER_SIZE", 4096)
	config.Audio.Channels = getEnvAsInt("AUDIO_CHANNELS", 2)
	config.Audio.SampleRate = getEnvAsInt("AUDIO_SAMPLE_RATE", 44100)
	config.Audio.BitsPerSample = getEnvAsInt("AUDIO_BITS_PER_SAMPLE", 16)

	return config, nil
}
