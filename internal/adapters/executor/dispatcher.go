package executor

import (
	"context"
	"errors"
	"fmt"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

var ErrHandlerNotRegistered = errors.New(
	"job handler not registered",
)

var requiredJobTypes = []job.Type{
	job.TypeSendEmail,
	job.TypeReportGeneration,
	job.TypeDataCleanup,
}

// Handler performs the operation belonging to one job type.
//
// Implementations should validate their type-specific payload and perform
// only their own background operation.
type Handler interface {
	Handle(ctx context.Context, entity *job.Job) error
}

// Dispatcher routes a job to the handler registered for its type.
type Dispatcher struct {
	handlers map[job.Type]Handler
}

var _ ports.JobExecutor = (*Dispatcher)(nil)

func NewDispatcher(
	handlers map[job.Type]Handler,
) (*Dispatcher, error) {
	copiedHandlers := make(
		map[job.Type]Handler,
		len(handlers),
	)

	for jobType, handler := range handlers {
		if !jobType.IsValid() {
			return nil, fmt.Errorf(
				"%w: %q",
				job.ErrInvalidType,
				jobType,
			)
		}

		if handler == nil {
			return nil, fmt.Errorf(
				"%w: nil handler for %s",
				ErrHandlerNotRegistered,
				jobType,
			)
		}

		copiedHandlers[jobType] = handler
	}

	for _, requiredType := range requiredJobTypes {
		if _, exists := copiedHandlers[requiredType]; !exists {
			return nil, fmt.Errorf(
				"%w: %s",
				ErrHandlerNotRegistered,
				requiredType,
			)
		}
	}

	return &Dispatcher{
		handlers: copiedHandlers,
	}, nil
}

func (dispatcher *Dispatcher) Execute(
	ctx context.Context,
	entity *job.Job,
) error {
	if entity == nil {
		return errors.New("job must not be nil")
	}

	handler, exists := dispatcher.handlers[entity.Type()]
	if !exists {
		return fmt.Errorf(
			"%w: %s",
			ErrHandlerNotRegistered,
			entity.Type(),
		)
	}

	if err := handler.Handle(ctx, entity); err != nil {
		return fmt.Errorf(
			"handle %s job %s: %w",
			entity.Type(),
			entity.ID(),
			err,
		)
	}

	return nil
}
