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

func TestJobLifecycleObserverLogsPersistedTransition(t *testing.T) {
	entity, err := job.New(job.NewParams{
		ID:        job.ID(loggingTestJobID),
		Type:      job.TypeDataCleanup,
		Payload:   json.RawMessage(`{"scope":"expired"}`),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create test job: %v", err)
	}

	var output bytes.Buffer
	observer, err := logging.NewJobLifecycleObserver(
		slog.New(slog.NewJSONHandler(&output, nil)),
	)
	if err != nil {
		t.Fatalf("create lifecycle observer: %v", err)
	}
	observer.StatusPersisted(context.Background(), entity, "")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("decode lifecycle log: %v", err)
	}

	if entry["msg"] != "job status persisted" {
		t.Fatalf("unexpected lifecycle message: %v", entry["msg"])
	}
	if entry["previous_status"] != "none" || entry["new_status"] != "pending" {
		t.Fatalf(
			"expected none -> pending transition, got %v -> %v",
			entry["previous_status"],
			entry["new_status"],
		)
	}
	if entry["job_id"] != loggingTestJobID || entry["job_type"] != "data_cleanup" {
		t.Fatalf("lifecycle log is missing job identity: %v", entry)
	}
}
