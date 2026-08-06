package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/taufMFI4430d/async-job-poc/internal/adapters/http/handler"
)

type readinessCheckerStub struct {
	err error
}

func (checker readinessCheckerStub) Ping(context.Context) error {
	return checker.err
}

func TestLiveReturnsOK(t *testing.T) {
	healthHandler := handler.NewHealthHandler(readinessCheckerStub{})

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
	healthHandler := handler.NewHealthHandler(readinessCheckerStub{})

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

func TestReadyReturnsServiceUnavailableWhenDependencyIsDown(t *testing.T) {
	healthHandler := handler.NewHealthHandler(readinessCheckerStub{
		err: errors.New("database unavailable"),
	})

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/ready",
		nil,
	)
	response := httptest.NewRecorder()

	healthHandler.Ready(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			response.Code,
		)
	}

	if !strings.Contains(response.Body.String(), `"status":"not_ready"`) {
		t.Fatalf(
			"unexpected response body: %s",
			response.Body.String(),
		)
	}
}

func TestReadyChecksAllDependencies(t *testing.T) {
	healthHandler := handler.NewHealthHandler(
		readinessCheckerStub{},
		readinessCheckerStub{
			err: errors.New("Redis unavailable"),
		},
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/ready",
		nil,
	)
	response := httptest.NewRecorder()

	healthHandler.Ready(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			response.Code,
		)
	}
}
