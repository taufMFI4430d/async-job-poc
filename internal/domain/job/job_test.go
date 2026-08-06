package job_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const validJobID = "a2201f2e-9320-4e63-a4ab-7447c7ba40dd"

func TestNewCreatesPendingJobWithDefaults(t *testing.T) {
	createdAt := time.Date(2026, time.August, 3, 10, 30, 0, 0, time.FixedZone("IST", 5*60*60+30*60))

	createdJob, err := job.New(job.NewParams{
		ID:        job.ID(validJobID),
		Type:      job.TypeSendEmail,
		Payload:   json.RawMessage(`{"to":"learner@example.com"}`),
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	if createdJob.ID() != job.ID(validJobID) {
		t.Errorf("expected ID %q, got %q", validJobID, createdJob.ID())
	}

	if createdJob.Type() != job.TypeSendEmail {
		t.Errorf("expected type %q, got %q", job.TypeSendEmail, createdJob.Type())
	}

	if createdJob.Status() != job.StatusPending {
		t.Errorf("expected status %q, got %q", job.StatusPending, createdJob.Status())
	}

	if createdJob.RetryCount() != 0 {
		t.Errorf("expected retry count 0, got %d", createdJob.RetryCount())
	}

	if createdJob.MaxRetries() != job.DefaultMaxRetries {
		t.Errorf("expected max retries %d, got %d", job.DefaultMaxRetries, createdJob.MaxRetries())
	}

	if !createdJob.CreatedAt().Equal(createdAt.UTC()) {
		t.Errorf("expected UTC creation time %v, got %v", createdAt.UTC(), createdJob.CreatedAt())
	}

	if !createdJob.UpdatedAt().Equal(createdAt.UTC()) {
		t.Errorf("expected update time %v, got %v", createdAt.UTC(), createdJob.UpdatedAt())
	}

	if _, exists := createdJob.LastError(); exists {
		t.Error("expected no last error")
	}

	if _, exists := createdJob.StartedAt(); exists {
		t.Error("expected no start time")
	}

	if _, exists := createdJob.CompletedAt(); exists {
		t.Error("expected no completion time")
	}
}

func TestNewAcceptsEverySupportedType(t *testing.T) {
	supportedTypes := []job.Type{
		job.TypeSendEmail,
		job.TypeReportGeneration,
		job.TypeDataCleanup,
	}

	for _, jobType := range supportedTypes {
		t.Run(jobType.String(), func(t *testing.T) {
			_, err := job.New(job.NewParams{
				ID:        job.ID(validJobID),
				Type:      jobType,
				Payload:   json.RawMessage(`{"value":"test"}`),
				CreatedAt: time.Now(),
			})
			if err != nil {
				t.Fatalf("New() rejected supported type %q: %v", jobType, err)
			}
		})
	}
}

func TestNewRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name          string
		params        job.NewParams
		expectedError error
	}{
		{
			name: "invalid ID",
			params: job.NewParams{
				ID:        "not-a-uuid",
				Type:      job.TypeSendEmail,
				Payload:   json.RawMessage(`{"value":"test"}`),
				CreatedAt: time.Now(),
			},
			expectedError: job.ErrInvalidID,
		},
		{
			name: "unsupported type",
			params: job.NewParams{
				ID:        job.ID(validJobID),
				Type:      "unknown",
				Payload:   json.RawMessage(`{"value":"test"}`),
				CreatedAt: time.Now(),
			},
			expectedError: job.ErrInvalidType,
		},
		{
			name: "missing payload",
			params: job.NewParams{
				ID:        job.ID(validJobID),
				Type:      job.TypeSendEmail,
				CreatedAt: time.Now(),
			},
			expectedError: job.ErrInvalidPayload,
		},
		{
			name: "malformed payload",
			params: job.NewParams{
				ID:        job.ID(validJobID),
				Type:      job.TypeSendEmail,
				Payload:   json.RawMessage(`{"value":`),
				CreatedAt: time.Now(),
			},
			expectedError: job.ErrInvalidPayload,
		},
		{
			name: "payload is an array",
			params: job.NewParams{
				ID:        job.ID(validJobID),
				Type:      job.TypeSendEmail,
				Payload:   json.RawMessage(`["value"]`),
				CreatedAt: time.Now(),
			},
			expectedError: job.ErrInvalidPayload,
		},
		{
			name: "empty object payload",
			params: job.NewParams{
				ID:        job.ID(validJobID),
				Type:      job.TypeSendEmail,
				Payload:   json.RawMessage(`{}`),
				CreatedAt: time.Now(),
			},
			expectedError: job.ErrInvalidPayload,
		},
		{
			name: "missing creation time",
			params: job.NewParams{
				ID:      job.ID(validJobID),
				Type:    job.TypeSendEmail,
				Payload: json.RawMessage(`{"value":"test"}`),
			},
			expectedError: job.ErrInvalidCreatedAt,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := job.New(test.params)
			if !errors.Is(err, test.expectedError) {
				t.Fatalf("expected error %v, got %v", test.expectedError, err)
			}
		})
	}
}

func TestPayloadIsCopiedAtDomainBoundary(t *testing.T) {
	originalPayload := json.RawMessage(`{"value":"original"}`)

	createdJob, err := job.New(job.NewParams{
		ID:        job.ID(validJobID),
		Type:      job.TypeDataCleanup,
		Payload:   originalPayload,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	originalPayload[10] = 'X'
	firstRead := createdJob.Payload()
	firstRead[10] = 'Y'

	if string(createdJob.Payload()) != `{"value":"original"}` {
		t.Fatalf("job payload was modified through an external byte slice: %s", createdJob.Payload())
	}
}

func TestParseTypeNormalizesInput(t *testing.T) {
	parsedType, err := job.ParseType("  SEND_EMAIL ")
	if err != nil {
		t.Fatalf("ParseType() returned an unexpected error: %v", err)
	}

	if parsedType != job.TypeSendEmail {
		t.Fatalf("expected %q, got %q", job.TypeSendEmail, parsedType)
	}
}

func TestParseStatusRejectsUnknownStatus(t *testing.T) {
	_, err := job.ParseStatus("queued")
	if !errors.Is(err, job.ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestJobLifecycleTransitionsToSuccess(t *testing.T) {
	createdAt := time.Date(
		2026,
		time.August,
		5,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	entity, err := job.New(job.NewParams{
		ID:        job.ID(validJobID),
		Type:      job.TypeSendEmail,
		Payload:   json.RawMessage(`{"to":"learner@example.com"}`),
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	processingAt := createdAt.Add(time.Second)
	if err := entity.MarkProcessing(processingAt); err != nil {
		t.Fatalf("mark processing: %v", err)
	}

	if entity.Status() != job.StatusProcessing {
		t.Fatalf(
			"expected processing status, got %s",
			entity.Status(),
		)
	}

	startedAt, exists := entity.StartedAt()
	if !exists || !startedAt.Equal(processingAt) {
		t.Fatalf(
			"expected started_at %v, got %v",
			processingAt,
			startedAt,
		)
	}

	completedAt := processingAt.Add(time.Second)
	if err := entity.MarkSuccess(completedAt); err != nil {
		t.Fatalf("mark success: %v", err)
	}

	if entity.Status() != job.StatusSuccess {
		t.Fatalf(
			"expected success status, got %s",
			entity.Status(),
		)
	}

	actualCompletedAt, exists := entity.CompletedAt()
	if !exists || !actualCompletedAt.Equal(completedAt) {
		t.Fatalf(
			"expected completed_at %v, got %v",
			completedAt,
			actualCompletedAt,
		)
	}
}

func TestJobLifecycleTransitionsToFailed(t *testing.T) {
	createdAt := time.Now().UTC()

	entity, err := job.New(job.NewParams{
		ID:        job.ID(validJobID),
		Type:      job.TypeReportGeneration,
		Payload:   json.RawMessage(`{"report":"monthly"}`),
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := entity.MarkProcessing(
		createdAt.Add(time.Second),
	); err != nil {
		t.Fatalf("mark processing: %v", err)
	}

	failedAt := createdAt.Add(2 * time.Second)
	if err := entity.MarkFailed(
		failedAt,
		" report generator unavailable ",
	); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	if entity.Status() != job.StatusFailed {
		t.Fatalf(
			"expected failed status, got %s",
			entity.Status(),
		)
	}

	lastError, exists := entity.LastError()
	if !exists {
		t.Fatal("expected a last error")
	}

	if lastError != "report generator unavailable" {
		t.Fatalf("unexpected last error: %q", lastError)
	}
}

func TestJobRejectsInvalidLifecycleTransition(t *testing.T) {
	createdAt := time.Now().UTC()

	entity, err := job.New(job.NewParams{
		ID:        job.ID(validJobID),
		Type:      job.TypeDataCleanup,
		Payload:   json.RawMessage(`{"scope":"expired"}`),
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	err = entity.MarkSuccess(createdAt.Add(time.Second))
	if !errors.Is(err, job.ErrInvalidTransition) {
		t.Fatalf(
			"expected ErrInvalidTransition, got %v",
			err,
		)
	}

	if entity.Status() != job.StatusPending {
		t.Fatalf(
			"invalid transition changed status to %s",
			entity.Status(),
		)
	}
}

func TestJobRejectsLifecycleTimeBeforeLastUpdate(t *testing.T) {
	createdAt := time.Now().UTC()

	entity, err := job.New(job.NewParams{
		ID:        job.ID(validJobID),
		Type:      job.TypeSendEmail,
		Payload:   json.RawMessage(`{"to":"learner@example.com"}`),
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	err = entity.MarkProcessing(createdAt.Add(-time.Second))
	if !errors.Is(err, job.ErrInvalidEventTime) {
		t.Fatalf(
			"expected ErrInvalidEventTime, got %v",
			err,
		)
	}
}
