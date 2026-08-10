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

const reportHandlerTestJobID = "602cf3a8-0bf2-4645-b966-b61696b2fb25"

func newReportTestLogger() *slog.Logger {
	return slog.New(
		slog.NewJSONHandler(io.Discard, nil),
	)
}

func newReportTestJob(
	t *testing.T,
	jobType job.Type,
	payload string,
) *job.Job {
	t.Helper()

	entity, err := job.New(job.NewParams{
		ID:        job.ID(reportHandlerTestJobID),
		Type:      jobType,
		Payload:   json.RawMessage(payload),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create test job: %v", err)
	}

	return entity
}

func TestReportGenerationHandlerCompletesValidJob(
	t *testing.T,
) {
	handler, err := executor.NewReportGenerationHandler(
		time.Millisecond,
		newReportTestLogger(),
	)
	if err != nil {
		t.Fatalf(
			"create report-generation handler: %v",
			err,
		)
	}

	entity := newReportTestJob(
		t,
		job.TypeReportGeneration,
		`{
			"report": "monthly_sales",
			"format": "PDF"
		}`,
	)

	if err := handler.Handle(
		context.Background(),
		entity,
	); err != nil {
		t.Fatalf(
			"handle report-generation job: %v",
			err,
		)
	}
}

func TestReportGenerationHandlerAcceptsSupportedFormats(
	t *testing.T,
) {
	formats := []string{
		"csv",
		"json",
		"pdf",
	}

	handler, err := executor.NewReportGenerationHandler(
		time.Millisecond,
		newReportTestLogger(),
	)
	if err != nil {
		t.Fatalf(
			"create report-generation handler: %v",
			err,
		)
	}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			entity := newReportTestJob(
				t,
				job.TypeReportGeneration,
				`{
					"report": "monthly_sales",
					"format": "`+format+`"
				}`,
			)

			if err := handler.Handle(
				context.Background(),
				entity,
			); err != nil {
				t.Fatalf(
					"format %s was rejected: %v",
					format,
					err,
				)
			}
		})
	}
}

func TestReportGenerationHandlerRejectsInvalidPayload(
	t *testing.T,
) {
	longReportName := strings.Repeat("a", 101)

	tests := []struct {
		name    string
		payload string
	}{
		{
			name: "missing report",
			payload: `{
				"format": "pdf"
			}`,
		},
		{
			name: "report name too long",
			payload: `{
				"report": "` + longReportName + `",
				"format": "pdf"
			}`,
		},
		{
			name: "missing format",
			payload: `{
				"report": "monthly_sales"
			}`,
		},
		{
			name: "unsupported format",
			payload: `{
				"report": "monthly_sales",
				"format": "xml"
			}`,
		},
		{
			name: "unknown field",
			payload: `{
				"report": "monthly_sales",
				"format": "pdf",
				"compress": true
			}`,
		},
	}

	handler, err := executor.NewReportGenerationHandler(
		time.Millisecond,
		newReportTestLogger(),
	)
	if err != nil {
		t.Fatalf(
			"create report-generation handler: %v",
			err,
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entity := newReportTestJob(
				t,
				job.TypeReportGeneration,
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

func TestReportGenerationHandlerRejectsUnexpectedType(
	t *testing.T,
) {
	handler, err := executor.NewReportGenerationHandler(
		time.Millisecond,
		newReportTestLogger(),
	)
	if err != nil {
		t.Fatalf(
			"create report-generation handler: %v",
			err,
		)
	}

	entity := newReportTestJob(
		t,
		job.TypeSendEmail,
		`{
			"to": "learner@example.com",
			"subject": "Report",
			"body": "Your report is ready."
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

func TestReportGenerationHandlerStopsOnCancellation(
	t *testing.T,
) {
	handler, err := executor.NewReportGenerationHandler(
		time.Hour,
		newReportTestLogger(),
	)
	if err != nil {
		t.Fatalf(
			"create report-generation handler: %v",
			err,
		)
	}

	entity := newReportTestJob(
		t,
		job.TypeReportGeneration,
		`{
			"report": "monthly_sales",
			"format": "pdf"
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

func TestReportGenerationHandlerRejectsNilJob(
	t *testing.T,
) {
	handler, err := executor.NewReportGenerationHandler(
		time.Millisecond,
		newReportTestLogger(),
	)
	if err != nil {
		t.Fatalf(
			"create report-generation handler: %v",
			err,
		)
	}

	err = handler.Handle(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error for nil job")
	}
}

func TestNewReportGenerationHandlerRejectsInvalidDuration(
	t *testing.T,
) {
	_, err := executor.NewReportGenerationHandler(
		0,
		newReportTestLogger(),
	)
	if err == nil {
		t.Fatal("expected an error for invalid duration")
	}
}

func TestNewReportGenerationHandlerRejectsNilLogger(
	t *testing.T,
) {
	_, err := executor.NewReportGenerationHandler(
		time.Millisecond,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for nil logger")
	}
}
