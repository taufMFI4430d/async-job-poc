package ports

import (
	"context"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

// JobLifecycleObserver observes status changes after they have been persisted.
// Observability is deliberately kept outside the domain model.
type JobLifecycleObserver interface {
	StatusPersisted(ctx context.Context, entity *job.Job, previousStatus job.Status)
}
