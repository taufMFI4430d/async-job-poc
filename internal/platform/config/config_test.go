package config_test

import (
	"testing"

	"github.com/taufMFI4430d/async-job-poc/internal/platform/config"
)

func TestLoadReturnsValidConfiguration(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("HTTP_ADDRESS", ":9090")
	t.Setenv("UI_ASSETS_DIR", "testdata/ui")
	t.Setenv("MYSQL_HOST", "localhost")
	t.Setenv("MYSQL_PORT", "3307")
	t.Setenv("MYSQL_DATABASE", "async_jobs_test")
	t.Setenv("MYSQL_USER", "test_user")
	t.Setenv("MYSQL_PASSWORD", "test_password")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_QUEUE_NAME", "test:jobs:pending")

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

	if cfg.UI.AssetsDirectory != "testdata/ui" {
		t.Errorf(
			"expected UI assets directory testdata/ui, got %s",
			cfg.UI.AssetsDirectory,
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

	if cfg.Redis.DB != 2 {
		t.Errorf(
			"expected Redis DB 2, got %d",
			cfg.Redis.DB,
		)
	}

	if cfg.Redis.QueueName != "test:jobs:pending" {
		t.Errorf(
			"expected Redis queue name %q, got %q",
			"test:jobs:pending",
			cfg.Redis.QueueName,
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

func TestLoadRejectsNegativeRedisDB(t *testing.T) {
	t.Setenv("MYSQL_HOST", "localhost")
	t.Setenv("MYSQL_DATABASE", "async_jobs_test")
	t.Setenv("MYSQL_USER", "test_user")
	t.Setenv("MYSQL_PASSWORD", "test_password")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("REDIS_DB", "-1")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected an error for a negative Redis DB")
	}
}
