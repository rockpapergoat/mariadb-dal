package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DBHost     string   // DB_HOST
	DBPort     string   // DB_PORT
	DBName     string   // DB_NAME
	DBUser     string   // DB_USER
	DBPassword string   // DB_PASSWORD
	APIKeys    []string // API_KEYS (comma-separated)
	ListenAddr string   // LISTEN_ADDR (default: :8080)
}

// Load reads configuration from environment variables and returns a Config.
// Returns an error if any required variable is missing or if API_KEYS is empty after splitting.
func Load() (*Config, error) {
	required := []string{"DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD", "API_KEYS"}
	for _, key := range required {
		if os.Getenv(key) == "" {
			return nil, fmt.Errorf("required environment variable %s is not set", key)
		}
	}

	apiKeys := splitAndTrim(os.Getenv("API_KEYS"), ",")
	if len(apiKeys) == 0 {
		return nil, fmt.Errorf("API_KEYS must contain at least one key")
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	return &Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBName:     os.Getenv("DB_NAME"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		APIKeys:    apiKeys,
		ListenAddr: listenAddr,
	}, nil
}

// splitAndTrim splits s by sep and removes empty entries.
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
