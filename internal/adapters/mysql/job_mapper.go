package mysql

import (
	"errors"
	"fmt"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

var errNilJob = errors.New("job must not be nil")

func jobRecordFromDomain(entity *job.Job) (jobRecord, error) {
	if entity == nil {
		return jobRecord{}, errNilJob
	}

	record := jobRecord{
		ID:         entity.ID().String(),
		Type:       entity.Type().String(),
		Status:     entity.Status().String(),
		Payload:    entity.Payload(),
		RetryCount: entity.RetryCount(),
		MaxRetries: entity.MaxRetries(),
		CreatedAt:  entity.CreatedAt(),
		UpdatedAt:  entity.UpdatedAt(),
	}

	if lastError, exists := entity.LastError(); exists {
		record.LastError = &lastError
	}

	if startedAt, exists := entity.StartedAt(); exists {
		record.StartedAt = &startedAt
	}

	if completedAt, exists := entity.CompletedAt(); exists {
		record.CompletedAt = &completedAt
	}

	return record, nil
}

func jobRecordToDomain(record jobRecord) (*job.Job, error) {
	jobID, err := job.ParseID(record.ID)
	if err != nil {
		return nil, fmt.Errorf("map persisted job ID: %w", err)
	}

	jobType, err := job.ParseType(record.Type)
	if err != nil {
		return nil, fmt.Errorf("map persisted job type: %w", err)
	}

	status, err := job.ParseStatus(record.Status)
	if err != nil {
		return nil, fmt.Errorf("map persisted job status: %w", err)
	}

	entity, err := job.Restore(job.RestoreParams{
		ID:          jobID,
		Type:        jobType,
		Status:      status,
		Payload:     record.Payload,
		RetryCount:  record.RetryCount,
		MaxRetries:  record.MaxRetries,
		LastError:   record.LastError,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
		StartedAt:   record.StartedAt,
		CompletedAt: record.CompletedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("restore persisted job: %w", err)
	}

	return entity, nil
}
