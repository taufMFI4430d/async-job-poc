package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

type ProcessJob struct {
	repository ports.JobRepository
	executor   ports.JobExecutor
	clock      ports.Clock
}

func NewProcessJob(
	repository ports.JobRepository,
	executor ports.JobExecutor,
	clock ports.Clock,
) (*ProcessJob, error) {
	if repository == nil {
		return nil, errors.New("job repository must not be nil")
	}

	if executor == nil {
		return nil, errors.New("job executor must not be nil")
	}

	if clock == nil {
		return nil, errors.New("clock must not be nil")
	}

	return &ProcessJob{
		repository: repository,
		executor:   executor,
		clock:      clock,
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

	entity, err := useCase.repository.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf(
			"load job %s: %w",
			jobID,
			err,
		)
	}

	if err := entity.MarkProcessing(useCase.clock.Now()); err != nil {
		return fmt.Errorf(
			"mark job %s as processing: %w",
			jobID,
			err,
		)
	}

	if err := useCase.repository.Update(ctx, entity); err != nil {
		return fmt.Errorf(
			"persist processing status for job %s: %w",
			jobID,
			err,
		)
	}

	if err := useCase.executor.Execute(ctx, entity); err != nil {
		return useCase.handleExecutionFailure(
			ctx,
			entity,
			err,
		)
	}

	if err := entity.MarkSuccess(useCase.clock.Now()); err != nil {
		return fmt.Errorf(
			"mark job %s as successful: %w",
			jobID,
			err,
		)
	}

	if err := useCase.repository.Update(ctx, entity); err != nil {
		return fmt.Errorf(
			"persist successful status for job %s: %w",
			jobID,
			err,
		)
	}

	return nil
}

func (useCase *ProcessJob) handleExecutionFailure(
	ctx context.Context,
	entity *job.Job,
	executionError error,
) error {
	jobID := entity.ID()

	if err := entity.MarkFailed(
		useCase.clock.Now(),
		executionError.Error(),
	); err != nil {
		return errors.Join(
			fmt.Errorf(
				"execute job %s: %w",
				jobID,
				executionError,
			),
			fmt.Errorf(
				"mark job %s as failed: %w",
				jobID,
				err,
			),
		)
	}

	if err := useCase.repository.Update(ctx, entity); err != nil {
		return errors.Join(
			fmt.Errorf(
				"execute job %s: %w",
				jobID,
				executionError,
			),
			fmt.Errorf(
				"persist failed status for job %s: %w",
				jobID,
				err,
			),
		)
	}

	return fmt.Errorf(
		"execute job %s: %w",
		jobID,
		executionError,
	)
}
