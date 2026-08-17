package usecase

import (
	"context"
	"errors"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

type noopJobLifecycleObserver struct{}

func (noopJobLifecycleObserver) StatusPersisted(
	context.Context,
	*job.Job,
	job.Status,
) {
}

func resolveLifecycleObserver(
	observers []ports.JobLifecycleObserver,
) (ports.JobLifecycleObserver, error) {
	switch len(observers) {
	case 0:
		return noopJobLifecycleObserver{}, nil
	case 1:
		if observers[0] == nil {
			return nil, errors.New("lifecycle observer must not be nil")
		}

		return observers[0], nil
	default:
		return nil, errors.New("only one lifecycle observer may be provided")
	}
}
