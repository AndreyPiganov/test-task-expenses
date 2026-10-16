package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL", "")
	t.Setenv("APP_TIMEZONE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, defaultHTTPAddr)
	}
	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, defaultDatabaseURL)
	}
	if cfg.TelegramBotToken != "" {
		t.Errorf("TelegramBotToken = %q, want empty", cfg.TelegramBotToken)
	}
	if cfg.LLMBaseURL != "" || cfg.LLMAPIKey != "" || cfg.LLMModel != "" {
		t.Error("LLM configuration must be empty by default")
	}
	if cfg.AppTimezone.String() != defaultTimezone {
		t.Errorf("AppTimezone = %q, want %q", cfg.AppTimezone, defaultTimezone)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("HTTP_ADDR", "127.0.0.1:9090")
	t.Setenv("DATABASE_URL", "postgres://custom")
	t.Setenv("TELEGRAM_BOT_TOKEN", "secret-token")
	t.Setenv("LLM_BASE_URL", "https://llm.example/v1")
	t.Setenv("LLM_API_KEY", "secret-key")
	t.Setenv("LLM_MODEL", "example-model")
	t.Setenv("APP_TIMEZONE", "UTC")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "127.0.0.1:9090")
	}
	if cfg.DatabaseURL != "postgres://custom" {
		t.Errorf("DatabaseURL = %q, want postgres://custom", cfg.DatabaseURL)
	}
	if cfg.TelegramBotToken != "secret-token" {
		t.Errorf("TelegramBotToken = %q, want configured token", cfg.TelegramBotToken)
	}
	if cfg.LLMBaseURL != "https://llm.example/v1" || cfg.LLMAPIKey != "secret-key" || cfg.LLMModel != "example-model" {
		t.Error("LLM configuration was not loaded from environment")
	}
	if cfg.AppTimezone.String() != "UTC" {
		t.Errorf("AppTimezone = %q, want UTC", cfg.AppTimezone)
	}
}

func TestLoadRejectsUnknownTimezone(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "Unknown/Timezone")

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned nil error for an unknown timezone")
	}
}
