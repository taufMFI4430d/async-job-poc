package config_test

import (
	"github.com/taufMFI4430d/async-job-poc/internal/platform/config"
	"testing"
)

func TestLoadReturnsValidConfiguration(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("HTTP_ADDRESS", ":9090")
	t.Setenv("MYSQL_HOST", "localhost")
	t.Setenv("MYSQL_PORT", "3307")
	t.Setenv("MYSQL_DATABASE", "async_jobs_test")
	t.Setenv("MYSQL_USER", "test_user")
	t.Setenv("MYSQL_PASSWORD", "test_password")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("REDIS_PORT", "6380")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.HTTP.Address != ":9090" {
		t.Errorf(
			"expected HTTP address :9090, got %s",
			cfg.HTTP.Address,
		)
	}

	if cfg.MySQL.Port != 3307 {
		t.Errorf(
			"expected MySQL port 3307, got %d",
			cfg.MySQL.Port,
		)
	}

	if cfg.Redis.Port != 6380 {
		t.Errorf(
			"expected Redis port 6380, got %d",
			cfg.Redis.Port,
		)
	}
}

func TestLoadRejectsInvalidMySQLPort(t *testing.T) {
	t.Setenv("MYSQL_PORT", "not-a-number")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected an error for an invalid MySQL port")
	}
}

func TestLoadRejectsMissingRequiredValues(t *testing.T) {
	t.Setenv("MYSQL_HOST", "")
	t.Setenv("MYSQL_DATABASE", "")
	t.Setenv("MYSQL_USER", "")
	t.Setenv("MYSQL_PASSWORD", "")
	t.Setenv("REDIS_HOST", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected an error for missing required configuration")
	}
}
