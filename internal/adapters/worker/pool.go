package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

// WorkerCount is fixed to five to match the POC Definition of Done.
const WorkerCount = 5

type Pool struct {
	dispatcher *Dispatcher
	workers    []*ProcessorWorker
	logger     *slog.Logger
}

func NewPool(
	consumer ports.JobConsumer,
	processor JobProcessor,
	logger *slog.Logger,
) (*Pool, error) {
	if consumer == nil {
		return nil, errors.New(
			"job consumer must not be nil",
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

	dispatcher, err := NewDispatcher(consumer, logger)
	if err != nil {
		return nil, fmt.Errorf(
			"create queue dispatcher: %w",
			err,
		)
	}

	workers := make(
		[]*ProcessorWorker,
		0,
		WorkerCount,
	)

	for workerID := 1; workerID <= WorkerCount; workerID++ {
		processingWorker, err := NewProcessorWorker(
			workerID,
			processor,
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"create processing worker %d: %w",
				workerID,
				err,
			)
		}

		workers = append(workers, processingWorker)
	}

	return &Pool{
		dispatcher: dispatcher,
		workers:    workers,
		logger:     logger,
	}, nil
}

func (pool *Pool) Run(ctx context.Context) error {
	// The unbuffered channel prevents the dispatcher from fetching work
	// faster than the processing workers can accept it.
	jobs := make(chan job.ID)

	pool.logger.Info(
		"worker pool started",
		slog.Int("worker_count", len(pool.workers)),
	)

	var workerGroup sync.WaitGroup

	workerErrors := make(
		chan error,
		len(pool.workers),
	)

	for _, processingWorker := range pool.workers {
		workerGroup.Add(1)

		go func(currentWorker *ProcessorWorker) {
			defer workerGroup.Done()

			if err := currentWorker.Run(ctx, jobs); err != nil {
				workerErrors <- err
			}
		}(processingWorker)
	}

	// Run blocks while Redis is being consumed. On cancellation,
	// Dispatcher.Run returns and closes the jobs channel.
	dispatcherError := pool.dispatcher.Run(ctx, jobs)

	// Wait until every processing goroutine has exited.
	workerGroup.Wait()
	close(workerErrors)

	var runErrors []error

	if dispatcherError != nil {
		runErrors = append(
			runErrors,
			fmt.Errorf(
				"run queue dispatcher: %w",
				dispatcherError,
			),
		)
	}

	for err := range workerErrors {
		runErrors = append(
			runErrors,
			fmt.Errorf(
				"run processing worker: %w",
				err,
			),
		)
	}

	if len(runErrors) > 0 {
		return errors.Join(runErrors...)
	}

	pool.logger.Info(
		"worker pool stopped gracefully",
		slog.Int("worker_count", len(pool.workers)),
	)

	return nil
}
