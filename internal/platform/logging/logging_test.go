package logging_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/taufMFI4430d/async-job-poc/internal/platform/logging"
)

func TestNewProducesStructuredJSON(t *testing.T) {
	var output bytes.Buffer

	logger, err := logging.New(
		&output,
		logging.Options{
			Level:       "info",
			Environment: "test",
			ServiceName: "test-service",
		},
	)
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	logger.Info("application started", "port", 8080)

	var entry map[string]any

	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["msg"] != "application started" {
		t.Errorf("expected message %q, got %v", "application started", entry["msg"])
	}

	if entry["level"] != "INFO" {
		t.Errorf("expected INFO level, got %v", entry["level"])
	}

	if entry["service"] != "test-service" {
		t.Errorf("expected test-service, got %v", entry["service"])
	}

	if entry["environment"] != "test" {
		t.Errorf("expected test environment, got %v", entry["environment"])
	}
}

func TestLoggerFiltersLowerLevels(t *testing.T) {
	var output bytes.Buffer

	logger, err := logging.New(
		&output,
		logging.Options{
			Level:       "warn",
			Environment: "test",
		},
	)
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	logger.Info("this should be filtered")

	if output.Len() != 0 {
		t.Fatalf("expected info log to be filtered, got %s", output.String())
	}
}

func TestNewRejectsInvalidLevel(t *testing.T) {
	var output bytes.Buffer

	_, err := logging.New(
		&output,
		logging.Options{
			Level:       "verbose",
			Environment: "test",
		},
	)

	if err == nil {
		t.Fatal("expected an error for an invalid log level")
	}
}
