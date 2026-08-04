package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

type GetJobInput struct {
	ID string
}

type GetJob struct {
	repository ports.JobRepository
}

func NewGetJob(repository ports.JobRepository) (*GetJob, error) {
	if repository == nil {
		return nil, errors.New("job repository must not be nil")
	}

	return &GetJob{repository: repository}, nil
}

func (useCase *GetJob) Execute(
	ctx context.Context,
	input GetJobInput,
) (*job.Job, error) {
	jobID, err := job.ParseID(input.ID)
	if err != nil {
		return nil, fmt.Errorf("validate job ID: %w", err)
	}

	entity, err := useCase.repository.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}

	return entity, nil
}
