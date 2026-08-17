package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

var (
	ErrJobRetryScheduled = errors.New(
		"job retry scheduled",
	)

	ErrJobRetriesExhausted = errors.New(
		"job retries exhausted",
	)
)

type ProcessJob struct {
	repository        ports.JobRepository
	jobQueue          ports.JobQueue
	executor          ports.JobExecutor
	clock             ports.Clock
	lifecycleObserver ports.JobLifecycleObserver
}

func NewProcessJob(
	repository ports.JobRepository,
	jobQueue ports.JobQueue,
	executor ports.JobExecutor,
	clock ports.Clock,
	lifecycleObservers ...ports.JobLifecycleObserver,
) (*ProcessJob, error) {
	if repository == nil {
		return nil, errors.New(
			"job repository must not be nil",
		)
	}

	if jobQueue == nil {
		return nil, errors.New(
			"job queue must not be nil",
		)
	}

	if executor == nil {
		return nil, errors.New(
			"job executor must not be nil",
		)
	}

	if clock == nil {
		return nil, errors.New(
			"clock must not be nil",
		)
	}
	lifecycleObserver, err := resolveLifecycleObserver(lifecycleObservers)
	if err != nil {
		return nil, err
	}

	return &ProcessJob{
		repository:        repository,
		jobQueue:          jobQueue,
		executor:          executor,
		clock:             clock,
		lifecycleObserver: lifecycleObserver,
	}, nil
}

func (useCase *ProcessJob) Execute(
	ctx context.Context,
	jobID job.ID,
) error {
	if !jobID.IsValid() {
		return fmt.Errorf(
			"validate job ID: %w: %q",
			job.ErrInvalidID,
			jobID,
		)
	}

	entity, err := useCase.repository.GetByID(
		ctx,
		jobID,
	)
	if err != nil {
		return fmt.Errorf(
			"load job %s: %w",
			jobID,
			err,
		)
	}

	previousStatus := entity.Status()
	if err := entity.MarkProcessing(
		useCase.clock.Now(),
	); err != nil {
		return fmt.Errorf(
			"mark job %s as processing: %w",
			jobID,
			err,
		)
	}

	if err := useCase.repository.Update(
		ctx,
		entity,
	); err != nil {
		return fmt.Errorf(
			"persist processing status for job %s: %w",
			jobID,
			err,
		)
	}
	useCase.lifecycleObserver.StatusPersisted(ctx, entity, previousStatus)

	if err := useCase.executor.Execute(
		ctx,
		entity,
	); err != nil {
		return useCase.handleExecutionFailure(
			ctx,
			entity,
			err,
		)
	}

	previousStatus = entity.Status()
	if err := entity.MarkSuccess(
		useCase.clock.Now(),
	); err != nil {
		return fmt.Errorf(
			"mark job %s as successful: %w",
			jobID,
			err,
		)
	}

	if err := useCase.repository.Update(
		ctx,
		entity,
	); err != nil {
		return fmt.Errorf(
			"persist successful status for job %s: %w",
			jobID,
			err,
		)
	}
	useCase.lifecycleObserver.StatusPersisted(ctx, entity, previousStatus)

	return nil
}

func (useCase *ProcessJob) handleExecutionFailure(
	ctx context.Context,
	entity *job.Job,
	executionError error,
) error {
	jobID := entity.ID()
	previousStatus := entity.Status()

	outcome, err := entity.RecordFailure(
		useCase.clock.Now(),
		executionError.Error(),
	)
	if err != nil {
		return errors.Join(
			fmt.Errorf(
				"execute job %s: %w",
				jobID,
				executionError,
			),
			fmt.Errorf(
				"record failure for job %s: %w",
				jobID,
				err,
			),
		)
	}

	// Persist pending/failed before publishing a retry. Otherwise,
	// another worker could receive the ID while MySQL still says processing.
	if err := useCase.repository.Update(
		ctx,
		entity,
	); err != nil {
		return errors.Join(
			fmt.Errorf(
				"execute job %s: %w",
				jobID,
				executionError,
			),
			fmt.Errorf(
				"persist failure state for job %s: %w",
				jobID,
				err,
			),
		)
	}
	useCase.lifecycleObserver.StatusPersisted(ctx, entity, previousStatus)

	switch outcome {
	case job.FailureOutcomeRetryScheduled:
		if err := useCase.jobQueue.Enqueue(
			ctx,
			jobID,
		); err != nil {
			return errors.Join(
				fmt.Errorf(
					"execute job %s: %w",
					jobID,
					executionError,
				),
				fmt.Errorf(
					"enqueue retry %d of %d for job %s: %w",
					entity.RetryCount(),
					entity.MaxRetries(),
					jobID,
					err,
				),
			)
		}

		return errors.Join(
			fmt.Errorf(
				"%w: job %s retry %d of %d",
				ErrJobRetryScheduled,
				jobID,
				entity.RetryCount(),
				entity.MaxRetries(),
			),
			fmt.Errorf(
				"execute job %s: %w",
				jobID,
				executionError,
			),
		)

	case job.FailureOutcomeTerminalFailure:
		return errors.Join(
			fmt.Errorf(
				"%w: job %s failed after %d retries",
				ErrJobRetriesExhausted,
				jobID,
				entity.RetryCount(),
			),
			fmt.Errorf(
				"execute job %s: %w",
				jobID,
				executionError,
			),
		)

	default:
		return fmt.Errorf(
			"job %s produced unknown failure outcome %q",
			jobID,
			outcome,
		)
	}
}
