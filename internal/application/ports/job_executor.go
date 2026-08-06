package ports

import (
	"context"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

// JobExecutor performs the actual work represented by a job.
//
// Implementations may send emails, generate reports, clean data,
// or dispatch the job to a type-specific handler.
type JobExecutor interface {
	Execute(ctx context.Context, entity *job.Job) error
}
