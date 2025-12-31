package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	TelegramToken    string
	TelegramChat     string
	TelegramThreadID *int

	HTTPPort string
	CronSpec string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		DBHost:           envOrDefault("DB_HOST", "localhost"),
		DBPort:           envOrDefault("DB_PORT", "5432"),
		DBUser:           envOrDefault("DB_USERNAME", "postgres"),
		DBPassword:       envOrDefault("DB_PASSWORD", "postgres"),
		DBName:           envOrDefault("DB_DATABASE", "ponisha"),
		DBSSLMode:        envOrDefault("DB_SSLMODE", "disable"),
		TelegramToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChat:     os.Getenv("TELEGRAM_CHAT_ID"),
		TelegramThreadID: nil,
		HTTPPort:         envOrDefault("HTTP_PORT", "3000"),
		CronSpec:         envOrDefault("SCRAPE_CRON", "*/7 * * * *"),
	}

	threadID, err := envOrIntPtr("TELEGRAM_CHAT_THREAD_ID")
	if err != nil {
		return cfg, err
	}
	cfg.TelegramThreadID = threadID

	if cfg.TelegramToken == "" || cfg.TelegramChat == "" {
		return cfg, errors.New("missing TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID")
	}

	if cfg.DBHost == "" || cfg.DBUser == "" || cfg.DBName == "" {
		return cfg, errors.New("missing database configuration")
	}

	return cfg, nil
}

func (c Config) PostgresDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func envOrIntPtr(key string) (*int, error) {
	val := os.Getenv(key)
	if val == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", key, err)
	}
	return &parsed, nil
}
