package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
	"gorm.io/gorm"
)

type JobRepository struct {
	db *gorm.DB
}

var _ ports.JobRepository = (*JobRepository)(nil)

func NewJobRepository(db *gorm.DB) (*JobRepository, error) {
	if db == nil {
		return nil, errors.New("GORM database must not be nil")
	}

	return &JobRepository{db: db}, nil
}

func (repository *JobRepository) Create(
	ctx context.Context,
	entity *job.Job,
) error {
	record, err := jobRecordFromDomain(entity)
	if err != nil {
		return fmt.Errorf("map job for creation: %w", err)
	}

	if err := repository.db.WithContext(ctx).Create(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("%w: %s", ports.ErrJobAlreadyExists, record.ID)
		}

		return fmt.Errorf("create job %s: %w", record.ID, err)
	}

	return nil
}

func (repository *JobRepository) GetByID(
	ctx context.Context,
	id job.ID,
) (*job.Job, error) {
	if !id.IsValid() {
		return nil, fmt.Errorf("%w: %q", job.ErrInvalidID, id)
	}

	var record jobRecord
	err := repository.db.WithContext(ctx).
		Where("id = ?", id.String()).
		First(&record).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", ports.ErrJobNotFound, id)
		}

		return nil, fmt.Errorf("get job %s: %w", id, err)
	}

	entity, err := jobRecordToDomain(record)
	if err != nil {
		return nil, fmt.Errorf("map job %s from persistence: %w", id, err)
	}

	return entity, nil
}
