package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/taufMFI4430d/async-job-poc/internal/adapters/http/handler"
)

func TestLiveReturnsOK(t *testing.T) {
	healthHandler := handler.NewHealthHandler()

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/live",
		nil,
	)
	response := httptest.NewRecorder()

	healthHandler.Live(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.Code,
		)
	}

	if !strings.Contains(response.Body.String(), `"status":"ok"`) {
		t.Fatalf(
			"unexpected response body: %s",
			response.Body.String(),
		)
	}
}

func TestReadyReturnsOK(t *testing.T) {
	healthHandler := handler.NewHealthHandler()

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/ready",
		nil,
	)
	response := httptest.NewRecorder()

	healthHandler.Ready(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.Code,
		)
	}

	if !strings.Contains(response.Body.String(), `"status":"ready"`) {
		t.Fatalf(
			"unexpected response body: %s",
			response.Body.String(),
		)
	}
}
