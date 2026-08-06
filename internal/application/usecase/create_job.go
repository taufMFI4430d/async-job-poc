package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

type CreateJobInput struct {
	Type    string
	Payload json.RawMessage
}

type CreateJob struct {
	repository  ports.JobRepository
	jobQueue    ports.JobQueue
	idGenerator ports.IDGenerator
	clock       ports.Clock
}

func NewCreateJob(
	repository ports.JobRepository,
	jobQueue ports.JobQueue,
	idGenerator ports.IDGenerator,
	clock ports.Clock,
) (*CreateJob, error) {
	if repository == nil {
		return nil, errors.New("job repository must not be nil")
	}

	if jobQueue == nil {
		return nil, errors.New("job queue must not be nil")
	}

	if idGenerator == nil {
		return nil, errors.New("ID generator must not be nil")
	}

	if clock == nil {
		return nil, errors.New("clock must not be nil")
	}

	return &CreateJob{
		repository:  repository,
		jobQueue:    jobQueue,
		idGenerator: idGenerator,
		clock:       clock,
	}, nil
}

func (useCase *CreateJob) Execute(
	ctx context.Context,
	input CreateJobInput,
) (*job.Job, error) {
	jobType, err := job.ParseType(input.Type)
	if err != nil {
		return nil, fmt.Errorf("validate job type: %w", err)
	}

	jobID, err := useCase.idGenerator.NewID()
	if err != nil {
		return nil, fmt.Errorf("generate job ID: %w", err)
	}

	entity, err := job.New(job.NewParams{
		ID:        jobID,
		Type:      jobType,
		Payload:   input.Payload,
		CreatedAt: useCase.clock.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	if err := useCase.repository.Create(ctx, entity); err != nil {
		return nil, fmt.Errorf("persist job: %w", err)
	}

	if err := useCase.jobQueue.Enqueue(ctx, entity.ID()); err != nil {
		return nil, fmt.Errorf(
			"publish job %s to queue: %w",
			entity.ID(),
			err,
		)
	}

	return entity, nil
}
