package logging

import (
	"context"
	"errors"
	"log/slog"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

// JobLifecycleObserver logs job transitions only after persistence succeeds.
type JobLifecycleObserver struct {
	logger *slog.Logger
}

func NewJobLifecycleObserver(logger *slog.Logger) (*JobLifecycleObserver, error) {
	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}

	return &JobLifecycleObserver{logger: logger}, nil
}

func (observer *JobLifecycleObserver) StatusPersisted(
	ctx context.Context,
	entity *job.Job,
	previousStatus job.Status,
) {
	previous := previousStatus.String()
	if previous == "" {
		previous = "none"
	}

	attributes := JobAttributes(entity)
	attributes = append(attributes,
		slog.String("previous_status", previous),
		slog.String("new_status", entity.Status().String()),
	)
	observer.logger.LogAttrs(ctx, slog.LevelInfo, "job status persisted", attributes...)
}

var _ ports.JobLifecycleObserver = (*JobLifecycleObserver)(nil)
