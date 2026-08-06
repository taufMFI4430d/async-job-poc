package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const dequeueRetryDelay = time.Second

// JobProcessor defines the operation the worker requires from the
// application layer.
//
// ProcessJob satisfies this interface without the worker depending on
// its concrete type.
type JobProcessor interface {
	Execute(ctx context.Context, jobID job.ID) error
}

type Worker struct {
	consumer  ports.JobConsumer
	processor JobProcessor
	logger    *slog.Logger
}

func New(
	consumer ports.JobConsumer,
	processor JobProcessor,
	logger *slog.Logger,
) (*Worker, error) {
	if consumer == nil {
		return nil, errors.New("job consumer must not be nil")
	}

	if processor == nil {
		return nil, errors.New("job processor must not be nil")
	}

	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}

	return &Worker{
		consumer:  consumer,
		processor: processor,
		logger:    logger,
	}, nil
}

// Run starts the blocking worker loop.
//
// Run does not create a goroutine. The caller decides which goroutine
// should execute the worker.
func (worker *Worker) Run(ctx context.Context) error {
	worker.logger.Info("worker started")

	for {
		jobID, err := worker.consumer.Dequeue(ctx)
		if err != nil {
			if contextFinished(ctx, err) {
				worker.logger.Info("worker stopped gracefully")
				return nil
			}

			worker.logger.Error(
				"failed to dequeue job",
				slog.Any("error", err),
			)

			if err := waitForRetry(ctx); err != nil {
				worker.logger.Info("worker stopped gracefully")
				return nil
			}

			continue
		}

		worker.logger.Info(
			"job dequeued",
			slog.String("job_id", jobID.String()),
		)

		worker.logger.Info(
			"job processing started",
			slog.String("job_id", jobID.String()),
		)

		if err := worker.processor.Execute(ctx, jobID); err != nil {
			worker.logger.Error(
				"job processing failed",
				slog.String("job_id", jobID.String()),
				slog.Any("error", err),
			)

			if ctx.Err() != nil {
				worker.logger.Info("worker stopped gracefully")
				return nil
			}

			continue
		}

		worker.logger.Info(
			"job processing completed",
			slog.String("job_id", jobID.String()),
		)
	}
}

func contextFinished(
	ctx context.Context,
	err error,
) bool {
	return ctx.Err() != nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

func waitForRetry(ctx context.Context) error {
	timer := time.NewTimer(dequeueRetryDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
