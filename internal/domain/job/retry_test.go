package job_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

func newRetryTestJob(
	t *testing.T,
	createdAt time.Time,
) *job.Job {
	t.Helper()

	entity, err := job.New(job.NewParams{
		ID:        job.ID(validJobID),
		Type:      job.TypeSendEmail,
		Payload:   json.RawMessage(`{"value":"test"}`),
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("create retry test job: %v", err)
	}

	return entity
}

func verifyRetryableFailure(
	t *testing.T,
	entity *job.Job,
	attempt int,
	outcome job.FailureOutcome,
) {
	t.Helper()

	if outcome != job.FailureOutcomeRetryScheduled {
		t.Fatalf(
			"attempt %d: expected retry outcome, got %s",
			attempt,
			outcome,
		)
	}

	if entity.Status() != job.StatusPending {
		t.Fatalf(
			"attempt %d: expected pending status, got %s",
			attempt,
			entity.Status(),
		)
	}

	if entity.RetryCount() != attempt {
		t.Fatalf(
			"attempt %d: expected retry count %d, got %d",
			attempt,
			attempt,
			entity.RetryCount(),
		)
	}

	if _, exists := entity.CompletedAt(); exists {
		t.Fatalf(
			"attempt %d: retryable job must not be completed",
			attempt,
		)
	}
}

func verifyTerminalFailure(
	t *testing.T,
	entity *job.Job,
	outcome job.FailureOutcome,
	eventTime time.Time,
) {
	t.Helper()

	if outcome != job.FailureOutcomeTerminalFailure {
		t.Fatalf(
			"expected terminal outcome, got %s",
			outcome,
		)
	}

	if entity.Status() != job.StatusFailed {
		t.Fatalf(
			"expected failed status, got %s",
			entity.Status(),
		)
	}

	if entity.RetryCount() != job.DefaultMaxRetries {
		t.Fatalf(
			"expected retry count %d, got %d",
			job.DefaultMaxRetries,
			entity.RetryCount(),
		)
	}

	completedAt, exists := entity.CompletedAt()
	if !exists {
		t.Fatal(
			"terminally failed job must have completed_at",
		)
	}

	if !completedAt.Equal(eventTime) {
		t.Fatalf(
			"expected completed_at %v, got %v",
			eventTime,
			completedAt,
		)
	}
}

func TestRecordFailureSchedulesThreeRetriesThenFails(
	t *testing.T,
) {
	createdAt := time.Date(
		2026,
		time.August,
		10,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	entity := newRetryTestJob(t, createdAt)
	eventTime := createdAt

	totalAttempts := job.DefaultMaxRetries + 1

	for attempt := 1; attempt <= totalAttempts; attempt++ {
		eventTime = eventTime.Add(time.Second)

		if err := entity.MarkProcessing(eventTime); err != nil {
			t.Fatalf(
				"attempt %d: mark processing: %v",
				attempt,
				err,
			)
		}

		eventTime = eventTime.Add(time.Second)

		outcome, err := entity.RecordFailure(
			eventTime,
			fmt.Sprintf("attempt %d failed", attempt),
		)
		if err != nil {
			t.Fatalf(
				"attempt %d: record failure: %v",
				attempt,
				err,
			)
		}

		if attempt <= job.DefaultMaxRetries {
			verifyRetryableFailure(t, entity, attempt, outcome)
			continue
		}

		verifyTerminalFailure(t, entity, outcome, eventTime)
	}

	lastError, exists := entity.LastError()
	if !exists {
		t.Fatal("expected final failure reason")
	}

	if lastError != "attempt 4 failed" {
		t.Fatalf(
			"unexpected final error: %q",
			lastError,
		)
	}
}

func TestMarkProcessingPreservesPreviousRetryError(
	t *testing.T,
) {
	createdAt := time.Now().UTC()
	entity := newRetryTestJob(t, createdAt)

	if err := entity.MarkProcessing(
		createdAt.Add(time.Second),
	); err != nil {
		t.Fatalf("mark first attempt processing: %v", err)
	}

	_, err := entity.RecordFailure(
		createdAt.Add(2*time.Second),
		"temporary provider failure",
	)
	if err != nil {
		t.Fatalf("record first failure: %v", err)
	}

	if err := entity.MarkProcessing(
		createdAt.Add(3 * time.Second),
	); err != nil {
		t.Fatalf("mark retry processing: %v", err)
	}

	lastError, exists := entity.LastError()
	if !exists {
		t.Fatal(
			"expected previous error to remain during retry",
		)
	}

	if lastError != "temporary provider failure" {
		t.Fatalf(
			"unexpected retry error: %q",
			lastError,
		)
	}
}

func TestRecordFailureRejectsPendingJob(t *testing.T) {
	createdAt := time.Now().UTC()
	entity := newRetryTestJob(t, createdAt)

	_, err := entity.RecordFailure(
		createdAt.Add(time.Second),
		"processing failed",
	)

	if !errors.Is(err, job.ErrInvalidTransition) {
		t.Fatalf(
			"expected ErrInvalidTransition, got %v",
			err,
		)
	}

	if entity.Status() != job.StatusPending {
		t.Fatalf(
			"invalid failure changed status to %s",
			entity.Status(),
		)
	}
}

func TestRecordFailureRejectsEmptyReason(t *testing.T) {
	createdAt := time.Now().UTC()
	entity := newRetryTestJob(t, createdAt)

	if err := entity.MarkProcessing(
		createdAt.Add(time.Second),
	); err != nil {
		t.Fatalf("mark processing: %v", err)
	}

	_, err := entity.RecordFailure(
		createdAt.Add(2*time.Second),
		"   ",
	)

	if !errors.Is(err, job.ErrInvalidFailureReason) {
		t.Fatalf(
			"expected ErrInvalidFailureReason, got %v",
			err,
		)
	}

	if entity.Status() != job.StatusProcessing {
		t.Fatalf(
			"invalid failure changed status to %s",
			entity.Status(),
		)
	}
}

func TestRecordFailureRejectsInvalidTime(t *testing.T) {
	createdAt := time.Now().UTC()
	entity := newRetryTestJob(t, createdAt)

	processingAt := createdAt.Add(time.Second)

	if err := entity.MarkProcessing(processingAt); err != nil {
		t.Fatalf("mark processing: %v", err)
	}

	_, err := entity.RecordFailure(
		createdAt,
		"processing failed",
	)

	if !errors.Is(err, job.ErrInvalidEventTime) {
		t.Fatalf(
			"expected ErrInvalidEventTime, got %v",
			err,
		)
	}
}
