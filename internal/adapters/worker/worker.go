package worker

import (
	"context"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

// JobProcessor defines the application operation required by workers.
//
// ProcessJob satisfies this interface without the worker adapter
// depending on its concrete implementation.
type JobProcessor interface {
	Execute(ctx context.Context, jobID job.ID) error
}
