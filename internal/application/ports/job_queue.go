package ports

import (
	"context"
	"errors"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

var ErrJobQueueUnavailable = errors.New("job queue unavailable")

// JobQueue defines the operation required by job-producing use cases.
type JobQueue interface {
	Enqueue(ctx context.Context, id job.ID) error
}

// JobConsumer defines the operation required by job-consuming workers.
type JobConsumer interface {
	Dequeue(ctx context.Context) (job.ID, error)
}
