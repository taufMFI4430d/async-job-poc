package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const dispatcherRetryDelay = time.Second

type Dispatcher struct {
	consumer ports.JobConsumer
	logger   *slog.Logger
}

func NewDispatcher(
	consumer ports.JobConsumer,
	logger *slog.Logger,
) (*Dispatcher, error) {
	if consumer == nil {
		return nil, errors.New("job consumer must not be nil")
	}

	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}

	return &Dispatcher{
		consumer: consumer,
		logger:   logger,
	}, nil
}

// Run continuously dequeues job IDs and forwards them to the jobs channel.
//
// The dispatcher owns the output channel and closes it when dispatching stops.
func (dispatcher *Dispatcher) Run(
	ctx context.Context,
	jobs chan<- job.ID,
) error {
	if jobs == nil {
		return errors.New("jobs channel must not be nil")
	}

	dispatcher.logger.Info("queue dispatcher started")

	defer func() {
		close(jobs)
		dispatcher.logger.Info(
			"queue dispatcher stopped gracefully",
		)
	}()

	for {
		jobID, err := dispatcher.consumer.Dequeue(ctx)
		if err != nil {
			if dispatcherContextFinished(ctx, err) {
				return nil
			}

			dispatcher.logger.Error(
				"failed to dequeue job",
				slog.Any("error", err),
			)

			if err := waitBeforeNextDequeue(ctx); err != nil {
				return nil
			}

			continue
		}

		dispatcher.logger.Info(
			"job dequeued from Redis",
			slog.String("job_id", jobID.String()),
		)

		select {
		case jobs <- jobID:
			dispatcher.logger.Debug(
				"job dispatched to worker pool",
				slog.String("job_id", jobID.String()),
			)

		case <-ctx.Done():
			return nil
		}
	}
}

func dispatcherContextFinished(
	ctx context.Context,
	err error,
) bool {
	return ctx.Err() != nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

func waitBeforeNextDequeue(
	ctx context.Context,
) error {
	timer := time.NewTimer(dispatcherRetryDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-timer.C:
		return nil
	}
}
