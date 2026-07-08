package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the GTM MCP Server.
type Config struct {
	// Server configuration
	Port    int
	BaseURL string

	// Logging
	LogLevel string

	// AllowedHosts lists additional trusted hostnames for dynamic base URL resolution.
	AllowedHosts []string

	// Service account authentication
	ServiceAccountAPIKey  string // SERVICE_ACCOUNT_API_KEY — bearer token required from clients
	ServiceAccountKeyJSON string // GOOGLE_SERVICE_ACCOUNT_KEY_JSON — service account key JSON content
	ServiceAccountKeyFile string // GOOGLE_SERVICE_ACCOUNT_KEY_FILE — path to service account key JSON file

	// TrustProxy enables trusting X-Forwarded-For for rate limiting.
	TrustProxy bool
}

// Load reads configuration from environment variables.
// It first attempts to load from .env file if present, then .env.local for overrides.
func Load() (*Config, error) {
	_ = godotenv.Load()
	_ = godotenv.Overload(".env.local")

	port := getEnvInt("PORT", 8080)

	cfg := &Config{
		Port:                  port,
		BaseURL:               getEnv("BASE_URL", fmt.Sprintf("http://localhost:%d", port)),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		AllowedHosts:          getEnvList("ALLOWED_HOSTS"),
		ServiceAccountAPIKey:  getEnv("SERVICE_ACCOUNT_API_KEY", ""),
		ServiceAccountKeyJSON: getEnv("GOOGLE_SERVICE_ACCOUNT_KEY_JSON", ""),
		ServiceAccountKeyFile: getEnv("GOOGLE_SERVICE_ACCOUNT_KEY_FILE", ""),
		TrustProxy:            getEnvBool("TRUST_PROXY", false),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvList(key string) []string {
	if value := os.Getenv(key); value != "" {
		var hosts []string
		for _, h := range strings.Split(value, ",") {
			if h = strings.TrimSpace(h); h != "" {
				hosts = append(hosts, h)
			}
		}
		return hosts
	}
	return nil
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
