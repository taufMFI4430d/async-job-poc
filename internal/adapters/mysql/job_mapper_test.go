package mysql

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const mapperTestJobID = "f9940817-d21b-4128-a03c-448edb24d5f7"

func TestJobRecordRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	original, err := job.New(job.NewParams{
		ID:        job.ID(mapperTestJobID),
		Type:      job.TypeReportGeneration,
		Payload:   json.RawMessage(`{"report":"monthly"}`),
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("create domain job: %v", err)
	}

	record, err := jobRecordFromDomain(original)
	if err != nil {
		t.Fatalf("map domain job to record: %v", err)
	}

	restored, err := jobRecordToDomain(record)
	if err != nil {
		t.Fatalf("map record to domain job: %v", err)
	}

	if restored.ID() != original.ID() {
		t.Errorf("expected ID %q, got %q", original.ID(), restored.ID())
	}

	if restored.Type() != original.Type() {
		t.Errorf("expected type %q, got %q", original.Type(), restored.Type())
	}

	if restored.Status() != original.Status() {
		t.Errorf("expected status %q, got %q", original.Status(), restored.Status())
	}

	if string(restored.Payload()) != string(original.Payload()) {
		t.Errorf("expected payload %s, got %s", original.Payload(), restored.Payload())
	}

	if !restored.CreatedAt().Equal(original.CreatedAt()) {
		t.Errorf("expected creation time %v, got %v", original.CreatedAt(), restored.CreatedAt())
	}
}

func TestJobRecordFromDomainRejectsNil(t *testing.T) {
	_, err := jobRecordFromDomain(nil)
	if !errors.Is(err, errNilJob) {
		t.Fatalf("expected errNilJob, got %v", err)
	}
}

func TestJobRecordToDomainRejectsCorruptStatus(t *testing.T) {
	_, err := jobRecordToDomain(jobRecord{
		ID:         mapperTestJobID,
		Type:       job.TypeSendEmail.String(),
		Status:     "unknown",
		Payload:    json.RawMessage(`{"value":"test"}`),
		RetryCount: 0,
		MaxRetries: job.DefaultMaxRetries,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if !errors.Is(err, job.ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}
