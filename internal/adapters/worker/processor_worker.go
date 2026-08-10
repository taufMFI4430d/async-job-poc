package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/taufMFI4430d/async-job-poc/internal/application/usecase"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

// ProcessorWorker receives job IDs from the internal jobs channel.
//
// It does not communicate directly with Redis or MySQL. ProcessJob handles
// the application workflow and persistence.
type ProcessorWorker struct {
	id        int
	processor JobProcessor
	logger    *slog.Logger
}

func NewProcessorWorker(
	id int,
	processor JobProcessor,
	logger *slog.Logger,
) (*ProcessorWorker, error) {
	if id <= 0 {
		return nil, errors.New(
			"worker ID must be greater than zero",
		)
	}

	if processor == nil {
		return nil, errors.New(
			"job processor must not be nil",
		)
	}

	if logger == nil {
		return nil, errors.New(
			"logger must not be nil",
		)
	}

	return &ProcessorWorker{
		id:        id,
		processor: processor,
		logger:    logger,
	}, nil
}

// Run waits for IDs from the jobs channel and processes them sequentially.
//
// Concurrency is achieved by running multiple ProcessorWorker instances
// against the same channel.
func (worker *ProcessorWorker) Run(
	ctx context.Context,
	jobs <-chan job.ID,
) error {
	if jobs == nil {
		return errors.New("jobs channel must not be nil")
	}

	logger := worker.logger.With(
		slog.Int("worker_id", worker.id),
	)

	logger.Info("processing worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("processing worker stopped gracefully")
			return nil

		case jobID, channelOpen := <-jobs:
			if !channelOpen {
				logger.Info(
					"processing worker stopped after jobs channel closed",
				)
				return nil
			}

			logger.Info(
				"job processing started",
				slog.String("job_id", jobID.String()),
			)

			if err := worker.processor.Execute(ctx, jobID); err != nil {
				switch {
				case errors.Is(
					err,
					usecase.ErrJobRetryScheduled,
				):
					logger.Warn(
						"job retry scheduled",
						slog.String(
							"job_id",
							jobID.String(),
						),
						slog.Any("error", err),
					)

				case errors.Is(
					err,
					usecase.ErrJobRetriesExhausted,
				):
					logger.Error(
						"job retries exhausted",
						slog.String(
							"job_id",
							jobID.String(),
						),
						slog.Any("error", err),
					)

				default:
					logger.Error(
						"job processing failed",
						slog.String(
							"job_id",
							jobID.String(),
						),
						slog.Any("error", err),
					)
				}

				if ctx.Err() != nil {
					logger.Info(
						"processing worker stopped gracefully",
					)
					return nil
				}

				continue
			}

			logger.Info(
				"job processing completed",
				slog.String("job_id", jobID.String()),
			)
		}
	}
}
