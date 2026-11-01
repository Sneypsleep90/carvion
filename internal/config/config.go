package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv      string
	DatabaseURL string
	AdaptersDir string
}

func Load() (Config, error) {
	appEnv := envOrDefault("APP_ENV", "development")
	if appEnv != "development" && appEnv != "test" && appEnv != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development, test, or production")
	}

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	if err := validateDatabaseURL(databaseURL); err != nil {
		return Config{}, err
	}

	return Config{
		AppEnv:      appEnv,
		DatabaseURL: databaseURL,
		AdaptersDir: envOrDefault("ADAPTERS_DIR", "adapters"),
	}, nil
}

type HTTPConfig struct {
	Host string
	Port int
}

func LoadHTTP() (HTTPConfig, error) {
	httpPortText := envOrDefault("HTTP_PORT", "8080")
	httpPort, err := strconv.Atoi(httpPortText)
	if err != nil || httpPort < 1 || httpPort > 65535 {
		return HTTPConfig{}, fmt.Errorf("HTTP_PORT must be an integer between 1 and 65535")
	}
	return HTTPConfig{
		Host: envOrDefault("HTTP_HOST", "127.0.0.1"),
		Port: httpPort,
	}, nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func validateDatabaseURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("DATABASE_URL must be a valid PostgreSQL URL")
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("DATABASE_URL must use the postgres or postgresql scheme")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("DATABASE_URL must include a host")
	}
	if strings.Trim(parsed.Path, "/") == "" {
		return fmt.Errorf("DATABASE_URL must include a database name")
	}
	return nil
}
