package config

import (
	"fmt"
	"os"
	"time"
	_ "time/tzdata"
)

const (
	defaultHTTPAddr    = ":8080"
	defaultTimezone    = "Europe/Moscow"
	defaultDatabaseURL = "postgres://expenses:expenses@localhost:5432/expenses?sslmode=disable"
)

type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	TelegramBotToken string
	LLMBaseURL       string
	LLMAPIKey        string
	LLMModel         string
	AppTimezone      *time.Location
}

func Load() (Config, error) {
	timezoneName := valueOrDefault("APP_TIMEZONE", defaultTimezone)
	timezone, err := time.LoadLocation(timezoneName)
	if err != nil {
		return Config{}, fmt.Errorf("load application timezone %q: %w", timezoneName, err)
	}

	return Config{
		HTTPAddr:         valueOrDefault("HTTP_ADDR", defaultHTTPAddr),
		DatabaseURL:      valueOrDefault("DATABASE_URL", defaultDatabaseURL),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		LLMBaseURL:       os.Getenv("LLM_BASE_URL"),
		LLMAPIKey:        os.Getenv("LLM_API_KEY"),
		LLMModel:         os.Getenv("LLM_MODEL"),
		AppTimezone:      timezone,
	}, nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
