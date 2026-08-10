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

const sendEmailHandlerTestJobID = "009a88b8-3020-419e-9046-20c715e7dd2c"

func newSendEmailTestLogger() *slog.Logger {
	return slog.New(
		slog.NewJSONHandler(io.Discard, nil),
	)
}

func newSendEmailTestJob(
	t *testing.T,
	jobType job.Type,
	payload string,
) *job.Job {
	t.Helper()

	entity, err := job.New(job.NewParams{
		ID:        job.ID(sendEmailHandlerTestJobID),
		Type:      jobType,
		Payload:   json.RawMessage(payload),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create test job: %v", err)
	}

	return entity
}

func TestSendEmailHandlerCompletesValidJob(
	t *testing.T,
) {
	handler, err := executor.NewSendEmailHandler(
		time.Millisecond,
		newSendEmailTestLogger(),
	)
	if err != nil {
		t.Fatalf("create send-email handler: %v", err)
	}

	entity := newSendEmailTestJob(
		t,
		job.TypeSendEmail,
		`{
			"to": "learner@example.com",
			"subject": "POC completed",
			"body": "The asynchronous job completed."
		}`,
	)

	if err := handler.Handle(
		context.Background(),
		entity,
	); err != nil {
		t.Fatalf("handle send-email job: %v", err)
	}
}

func TestSendEmailHandlerRejectsInvalidPayload(
	t *testing.T,
) {
	longSubject := strings.Repeat(
		"a",
		201,
	)

	tests := []struct {
		name    string
		payload string
	}{
		{
			name: "missing recipient",
			payload: `{
				"subject": "Welcome",
				"body": "Hello"
			}`,
		},
		{
			name: "invalid recipient",
			payload: `{
				"to": "not-an-email",
				"subject": "Welcome",
				"body": "Hello"
			}`,
		},
		{
			name: "missing subject",
			payload: `{
				"to": "learner@example.com",
				"body": "Hello"
			}`,
		},
		{
			name: "subject too long",
			payload: `{
				"to": "learner@example.com",
				"subject": "` + longSubject + `",
				"body": "Hello"
			}`,
		},
		{
			name: "missing body",
			payload: `{
				"to": "learner@example.com",
				"subject": "Welcome"
			}`,
		},
		{
			name: "unknown field",
			payload: `{
				"to": "learner@example.com",
				"subject": "Welcome",
				"body": "Hello",
				"priority": "high"
			}`,
		},
	}

	handler, err := executor.NewSendEmailHandler(
		time.Millisecond,
		newSendEmailTestLogger(),
	)
	if err != nil {
		t.Fatalf("create send-email handler: %v", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entity := newSendEmailTestJob(
				t,
				job.TypeSendEmail,
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

func TestSendEmailHandlerRejectsUnexpectedJobType(
	t *testing.T,
) {
	handler, err := executor.NewSendEmailHandler(
		time.Millisecond,
		newSendEmailTestLogger(),
	)
	if err != nil {
		t.Fatalf("create send-email handler: %v", err)
	}

	entity := newSendEmailTestJob(
		t,
		job.TypeReportGeneration,
		`{"report":"monthly"}`,
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

func TestSendEmailHandlerStopsWhenContextIsCancelled(
	t *testing.T,
) {
	handler, err := executor.NewSendEmailHandler(
		time.Hour,
		newSendEmailTestLogger(),
	)
	if err != nil {
		t.Fatalf("create send-email handler: %v", err)
	}

	entity := newSendEmailTestJob(
		t,
		job.TypeSendEmail,
		`{
			"to": "learner@example.com",
			"subject": "Welcome",
			"body": "Hello"
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

func TestSendEmailHandlerRejectsNilJob(t *testing.T) {
	handler, err := executor.NewSendEmailHandler(
		time.Millisecond,
		newSendEmailTestLogger(),
	)
	if err != nil {
		t.Fatalf("create send-email handler: %v", err)
	}

	err = handler.Handle(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error for nil job")
	}
}

func TestNewSendEmailHandlerRejectsInvalidDuration(
	t *testing.T,
) {
	_, err := executor.NewSendEmailHandler(
		0,
		newSendEmailTestLogger(),
	)
	if err == nil {
		t.Fatal("expected an error for invalid duration")
	}
}

func TestNewSendEmailHandlerRejectsNilLogger(
	t *testing.T,
) {
	_, err := executor.NewSendEmailHandler(
		time.Millisecond,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for nil logger")
	}
}
