package ports

import (
	"context"
	"errors"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

var (
	ErrJobNotFound      = errors.New("job not found")
	ErrJobAlreadyExists = errors.New("job already exists")
)

// JobRepository defines the persistence operations required by job use cases.
// Infrastructure adapters, such as the GORM repository, implement this port.
type JobRepository interface {
	Create(ctx context.Context, entity *job.Job) error
	GetByID(ctx context.Context, id job.ID) (*job.Job, error)
	Update(ctx context.Context, entity *job.Job) error
}
