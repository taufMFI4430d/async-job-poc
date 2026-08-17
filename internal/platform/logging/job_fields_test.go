package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/logging"
)

const loggingTestJobID = "da08bd50-4150-4ff6-b51e-7b3f892df414"

func TestJobAttributesProduceConsistentFields(
	t *testing.T,
) {
	entity, err := job.New(job.NewParams{
		ID:   job.ID(loggingTestJobID),
		Type: job.TypeReportGeneration,
		Payload: json.RawMessage(
			`{"report":"monthly","format":"pdf"}`,
		),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create test job: %v", err)
	}

	if err := entity.MarkProcessing(
		time.Now().UTC().Add(time.Second),
	); err != nil {
		t.Fatalf("mark job processing: %v", err)
	}

	var output bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(&output, nil),
	)

	logger.LogAttrs(
		context.Background(),
		slog.LevelInfo,
		"job lifecycle test",
		logging.JobAttributes(entity)...,
	)

	var entry map[string]any

	if err := json.Unmarshal(
		output.Bytes(),
		&entry,
	); err != nil {
		t.Fatalf("decode log entry: %v", err)
	}

	expectedStrings := map[string]string{
		"job_id":     loggingTestJobID,
		"job_type":   "report_generation",
		"job_status": "processing",
	}

	for field, expected := range expectedStrings {
		if entry[field] != expected {
			t.Fatalf(
				"expected %s=%q, got %v",
				field,
				expected,
				entry[field],
			)
		}
	}

	if entry["retry_count"] != float64(0) {
		t.Fatalf(
			"expected retry_count=0, got %v",
			entry["retry_count"],
		)
	}

	if entry["max_retries"] != float64(3) {
		t.Fatalf(
			"expected max_retries=3, got %v",
			entry["max_retries"],
		)
	}
}

func TestJobAttributesAcceptNilJob(t *testing.T) {
	attributes := logging.JobAttributes(nil)

	if attributes != nil {
		t.Fatalf(
			"expected nil attributes, got %v",
			attributes,
		)
	}
}
