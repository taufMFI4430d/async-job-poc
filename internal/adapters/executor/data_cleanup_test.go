package executor_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/adapters/executor"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const dataCleanupHandlerTestJobID = "39faf07b-ad13-4788-8329-79cb3b55726b"

func newDataCleanupTestLogger() *slog.Logger {
	return slog.New(
		slog.NewJSONHandler(io.Discard, nil),
	)
}

func newDataCleanupTestJob(
	t *testing.T,
	jobType job.Type,
	payload string,
) *job.Job {
	t.Helper()

	entity, err := job.New(job.NewParams{
		ID:        job.ID(dataCleanupHandlerTestJobID),
		Type:      jobType,
		Payload:   json.RawMessage(payload),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create test job: %v", err)
	}

	return entity
}

func TestDataCleanupHandlerCompletesValidJob(
	t *testing.T,
) {
	handler, err := executor.NewDataCleanupHandler(
		time.Millisecond,
		newDataCleanupTestLogger(),
	)
	if err != nil {
		t.Fatalf("create data-cleanup handler: %v", err)
	}

	entity := newDataCleanupTestJob(
		t,
		job.TypeDataCleanup,
		`{
			"scope": "expired_sessions",
			"older_than_days": 30
		}`,
	)

	if err := handler.Handle(
		context.Background(),
		entity,
	); err != nil {
		t.Fatalf("handle data-cleanup job: %v", err)
	}
}

func TestDataCleanupHandlerRejectsInvalidPayload(
	t *testing.T,
) {
	longScope := strings.Repeat("a", 101)

	tests := []struct {
		name    string
		payload string
	}{
		{
			name: "missing scope",
			payload: `{
				"older_than_days": 30
			}`,
		},
		{
			name: "scope too long",
			payload: `{
				"scope": "` + longScope + `",
				"older_than_days": 30
			}`,
		},
		{
			name: "missing age",
			payload: `{
				"scope": "expired_sessions"
			}`,
		},
		{
			name: "negative age",
			payload: `{
				"scope": "expired_sessions",
				"older_than_days": -1
			}`,
		},
		{
			name: "age exceeds limit",
			payload: `{
				"scope": "expired_sessions",
				"older_than_days": 3651
			}`,
		},
		{
			name: "unknown field",
			payload: `{
				"scope": "expired_sessions",
				"older_than_days": 30,
				"force": true
			}`,
		},
	}

	handler, err := executor.NewDataCleanupHandler(
		time.Millisecond,
		newDataCleanupTestLogger(),
	)
	if err != nil {
		t.Fatalf("create data-cleanup handler: %v", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entity := newDataCleanupTestJob(
				t,
				job.TypeDataCleanup,
				test.payload,
			)

			err := handler.Handle(
				context.Background(),
				entity,
			)

			if !errors.Is(
				err,
				executor.ErrInvalidHandlerPayload,
			) {
				t.Fatalf(
					"expected ErrInvalidHandlerPayload, got %v",
					err,
				)
			}
		})
	}
}

func TestDataCleanupHandlerRejectsUnexpectedType(
	t *testing.T,
) {
	handler, err := executor.NewDataCleanupHandler(
		time.Millisecond,
		newDataCleanupTestLogger(),
	)
	if err != nil {
		t.Fatalf("create data-cleanup handler: %v", err)
	}

	entity := newDataCleanupTestJob(
		t,
		job.TypeReportGeneration,
		`{
			"report": "monthly_sales",
			"format": "pdf"
		}`,
	)

	err = handler.Handle(
		context.Background(),
		entity,
	)

	if !errors.Is(
		err,
		executor.ErrUnexpectedJobType,
	) {
		t.Fatalf(
			"expected ErrUnexpectedJobType, got %v",
			err,
		)
	}
}

func TestDataCleanupHandlerStopsOnCancellation(
	t *testing.T,
) {
	handler, err := executor.NewDataCleanupHandler(
		time.Hour,
		newDataCleanupTestLogger(),
	)
	if err != nil {
		t.Fatalf("create data-cleanup handler: %v", err)
	}

	entity := newDataCleanupTestJob(
		t,
		job.TypeDataCleanup,
		`{
			"scope": "expired_sessions",
			"older_than_days": 30
		}`,
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	err = handler.Handle(ctx, entity)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"expected context cancellation, got %v",
			err,
		)
	}
}

func TestDataCleanupHandlerRejectsNilJob(
	t *testing.T,
) {
	handler, err := executor.NewDataCleanupHandler(
		time.Millisecond,
		newDataCleanupTestLogger(),
	)
	if err != nil {
		t.Fatalf("create data-cleanup handler: %v", err)
	}

	err = handler.Handle(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error for nil job")
	}
}

func TestNewDataCleanupHandlerRejectsInvalidDuration(
	t *testing.T,
) {
	_, err := executor.NewDataCleanupHandler(
		0,
		newDataCleanupTestLogger(),
	)
	if err == nil {
		t.Fatal("expected an error for invalid duration")
	}
}

func TestNewDataCleanupHandlerRejectsNilLogger(
	t *testing.T,
) {
	_, err := executor.NewDataCleanupHandler(
		time.Millisecond,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for nil logger")
	}
}
