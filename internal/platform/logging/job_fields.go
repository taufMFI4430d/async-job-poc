package logging

import (
	"log/slog"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

func JobIDAttribute(jobID job.ID) slog.Attr {
	return slog.String("job_id", jobID.String())
}

func JobAttributes(entity *job.Job) []slog.Attr {
	if entity == nil {
		return nil
	}

	return []slog.Attr{
		JobIDAttribute(entity.ID()),
		slog.String(
			"job_type",
			entity.Type().String(),
		),
		slog.String(
			"job_status",
			entity.Status().String(),
		),
		slog.Int(
			"retry_count",
			entity.RetryCount(),
		),
		slog.Int(
			"max_retries",
			entity.MaxRetries(),
		),
	}
}

func ErrorAttribute(err error) slog.Attr {
	return slog.Any("error", err)
}

func RequestIDAttribute(requestID string) slog.Attr {
	return slog.String("request_id", requestID)
}
