package executor

import (
	"context"
	"errors"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

// Simulated temporarily represents the real job-type executors
// that will be introduced on Day 6.
type Simulated struct {
	processingDuration time.Duration
}

var _ ports.JobExecutor = (*Simulated)(nil)

func NewSimulated(
	processingDuration time.Duration,
) (*Simulated, error) {
	if processingDuration <= 0 {
		return nil, errors.New(
			"processing duration must be greater than zero",
		)
	}

	return &Simulated{
		processingDuration: processingDuration,
	}, nil
}

func (executor *Simulated) Execute(
	ctx context.Context,
	entity *job.Job,
) error {
	if entity == nil {
		return errors.New("job must not be nil")
	}

	timer := time.NewTimer(executor.processingDuration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
