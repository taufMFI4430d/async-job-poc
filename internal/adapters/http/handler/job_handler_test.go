package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/adapters/http/handler"
	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/application/usecase"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const handlerTestJobID = "92d21dbf-2f45-4743-a2a4-3089e60e36b3"

type createJobExecutorStub struct {
	execute func(context.Context, usecase.CreateJobInput) (*job.Job, error)
}

func (executor createJobExecutorStub) Execute(
	ctx context.Context,
	input usecase.CreateJobInput,
) (*job.Job, error) {
	return executor.execute(ctx, input)
}

type getJobExecutorStub struct {
	execute func(context.Context, usecase.GetJobInput) (*job.Job, error)
}

func (executor getJobExecutorStub) Execute(
	ctx context.Context,
	input usecase.GetJobInput,
) (*job.Job, error) {
	return executor.execute(ctx, input)
}

func TestJobHandlerCreateReturnsAcceptedJob(t *testing.T) {
	expected := newHandlerTestJob(t)

	createExecutor := createJobExecutorStub{
		execute: func(_ context.Context, input usecase.CreateJobInput) (*job.Job, error) {
			if input.Type != job.TypeSendEmail.String() {
				t.Errorf("expected type %q, got %q", job.TypeSendEmail, input.Type)
			}

			return expected, nil
		},
	}

	jobHandler := newTestJobHandler(t, createExecutor, successfulGetExecutor(expected))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/jobs",
		strings.NewReader(`{"type":"send_email","payload":{"to":"learner@example.com"}}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	jobHandler.Create(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, response.Code)
	}

	expectedLocation := "/api/v1/jobs/" + handlerTestJobID
	if response.Header().Get("Location") != expectedLocation {
		t.Errorf("expected Location %q, got %q", expectedLocation, response.Header().Get("Location"))
	}

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["id"] != handlerTestJobID {
		t.Errorf("expected job ID %q, got %v", handlerTestJobID, body["id"])
	}

	if body["status"] != job.StatusPending.String() {
		t.Errorf("expected pending status, got %v", body["status"])
	}
}

func TestJobHandlerCreateRejectsMalformedJSON(t *testing.T) {
	jobHandler := newTestJobHandler(
		t,
		createJobExecutorStub{execute: unexpectedCreateCall(t)},
		successfulGetExecutor(newHandlerTestJob(t)),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/jobs",
		strings.NewReader(`{"type":`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	jobHandler.Create(response, request)

	assertAPIError(t, response, http.StatusBadRequest, "invalid_request")
}

func TestJobHandlerCreateRejectsUnknownRequestField(t *testing.T) {
	jobHandler := newTestJobHandler(
		t,
		createJobExecutorStub{execute: unexpectedCreateCall(t)},
		successfulGetExecutor(newHandlerTestJob(t)),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/jobs",
		strings.NewReader(`{"type":"send_email","payload":{"value":"test"},"priority":1}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	jobHandler.Create(response, request)

	assertAPIError(t, response, http.StatusBadRequest, "invalid_request")
}

func TestJobHandlerCreateRequiresJSONContentType(t *testing.T) {
	jobHandler := newTestJobHandler(
		t,
		createJobExecutorStub{execute: unexpectedCreateCall(t)},
		successfulGetExecutor(newHandlerTestJob(t)),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/jobs",
		strings.NewReader(`{"type":"send_email","payload":{"value":"test"}}`),
	)
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()

	jobHandler.Create(response, request)

	assertAPIError(t, response, http.StatusUnsupportedMediaType, "unsupported_media_type")
}

func TestJobHandlerCreateMapsDomainValidationError(t *testing.T) {
	createExecutor := createJobExecutorStub{
		execute: func(context.Context, usecase.CreateJobInput) (*job.Job, error) {
			return nil, job.ErrInvalidPayload
		},
	}
	jobHandler := newTestJobHandler(t, createExecutor, successfulGetExecutor(newHandlerTestJob(t)))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/jobs",
		strings.NewReader(`{"type":"send_email","payload":{}}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	jobHandler.Create(response, request)

	assertAPIError(t, response, http.StatusBadRequest, "validation_error")
}

func TestJobHandlerGetReturnsPersistedJob(t *testing.T) {
	expected := newHandlerTestJob(t)
	jobHandler := newTestJobHandler(
		t,
		createJobExecutorStub{execute: unexpectedCreateCall(t)},
		successfulGetExecutor(expected),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+handlerTestJobID, nil)
	request.SetPathValue("jobID", handlerTestJobID)
	response := httptest.NewRecorder()

	jobHandler.GetByID(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["id"] != handlerTestJobID {
		t.Errorf("expected job ID %q, got %v", handlerTestJobID, body["id"])
	}
}

func TestJobHandlerGetMapsNotFoundError(t *testing.T) {
	getExecutor := getJobExecutorStub{
		execute: func(context.Context, usecase.GetJobInput) (*job.Job, error) {
			return nil, ports.ErrJobNotFound
		},
	}
	jobHandler := newTestJobHandler(
		t,
		createJobExecutorStub{execute: unexpectedCreateCall(t)},
		getExecutor,
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+handlerTestJobID, nil)
	request.SetPathValue("jobID", handlerTestJobID)
	response := httptest.NewRecorder()

	jobHandler.GetByID(response, request)

	assertAPIError(t, response, http.StatusNotFound, "job_not_found")
}

func TestJobHandlerDoesNotExposeInternalError(t *testing.T) {
	getExecutor := getJobExecutorStub{
		execute: func(context.Context, usecase.GetJobInput) (*job.Job, error) {
			return nil, errors.New("secret database details")
		},
	}
	jobHandler := newTestJobHandler(
		t,
		createJobExecutorStub{execute: unexpectedCreateCall(t)},
		getExecutor,
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+handlerTestJobID, nil)
	request.SetPathValue("jobID", handlerTestJobID)
	response := httptest.NewRecorder()

	jobHandler.GetByID(response, request)

	assertAPIError(t, response, http.StatusInternalServerError, "internal_error")
	if strings.Contains(response.Body.String(), "database") {
		t.Fatalf("response exposed internal error: %s", response.Body.String())
	}
}

func newTestJobHandler(
	t *testing.T,
	createExecutor createJobExecutorStub,
	getExecutor getJobExecutorStub,
) *handler.JobHandler {
	t.Helper()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	jobHandler, err := handler.NewJobHandler(createExecutor, getExecutor, logger)
	if err != nil {
		t.Fatalf("NewJobHandler() returned an unexpected error: %v", err)
	}

	return jobHandler
}

func newHandlerTestJob(t *testing.T) *job.Job {
	t.Helper()

	entity, err := job.New(job.NewParams{
		ID:        job.ID(handlerTestJobID),
		Type:      job.TypeSendEmail,
		Payload:   json.RawMessage(`{"to":"learner@example.com"}`),
		CreatedAt: time.Date(2026, time.August, 3, 15, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create test job: %v", err)
	}

	return entity
}

func successfulGetExecutor(entity *job.Job) getJobExecutorStub {
	return getJobExecutorStub{
		execute: func(context.Context, usecase.GetJobInput) (*job.Job, error) {
			return entity, nil
		},
	}
}

func unexpectedCreateCall(t *testing.T) func(
	context.Context,
	usecase.CreateJobInput,
) (*job.Job, error) {
	t.Helper()

	return func(context.Context, usecase.CreateJobInput) (*job.Job, error) {
		t.Fatal("create-job use case should not be called")
		return nil, nil
	}
}

func assertAPIError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	expectedStatus int,
	expectedCode string,
) {
	t.Helper()

	if response.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, response.Code)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if body.Error.Code != expectedCode {
		t.Fatalf("expected error code %q, got %q", expectedCode, body.Error.Code)
	}
}
