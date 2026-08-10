package executor_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/adapters/executor"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const dispatcherExecutorTestJobID = "502de2e3-f975-4198-989f-832e34cc8572"

type jobHandlerStub struct {
	handle func(context.Context, *job.Job) error
}

func (handler jobHandlerStub) Handle(
	ctx context.Context,
	entity *job.Job,
) error {
	if handler.handle == nil {
		return nil
	}

	return handler.handle(ctx, entity)
}

func completeHandlerRegistry(
	handler executor.Handler,
) map[job.Type]executor.Handler {
	return map[job.Type]executor.Handler{
		job.TypeSendEmail:        handler,
		job.TypeReportGeneration: handler,
		job.TypeDataCleanup:      handler,
	}
}

func newDispatcherTestJob(
	t *testing.T,
	jobType job.Type,
) *job.Job {
	t.Helper()

	entity, err := job.New(job.NewParams{
		ID:        job.ID(dispatcherExecutorTestJobID),
		Type:      jobType,
		Payload:   json.RawMessage(`{"value":"test"}`),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create test job: %v", err)
	}

	return entity
}

func TestDispatcherRoutesEverySupportedJobType(
	t *testing.T,
) {
	supportedTypes := []job.Type{
		job.TypeSendEmail,
		job.TypeReportGeneration,
		job.TypeDataCleanup,
	}

	for _, expectedType := range supportedTypes {
		t.Run(expectedType.String(), func(t *testing.T) {
			var receivedType job.Type

			handlers := make(
				map[job.Type]executor.Handler,
				len(supportedTypes),
			)

			for _, registeredType := range supportedTypes {
				currentType := registeredType

				handlers[currentType] = jobHandlerStub{
					handle: func(
						_ context.Context,
						entity *job.Job,
					) error {
						if currentType == expectedType {
							receivedType = entity.Type()
						}

						return nil
					},
				}
			}

			dispatcher, err := executor.NewDispatcher(
				handlers,
			)
			if err != nil {
				t.Fatalf(
					"create dispatcher: %v",
					err,
				)
			}

			entity := newDispatcherTestJob(
				t,
				expectedType,
			)

			if err := dispatcher.Execute(
				context.Background(),
				entity,
			); err != nil {
				t.Fatalf(
					"execute dispatched job: %v",
					err,
				)
			}

			if receivedType != expectedType {
				t.Fatalf(
					"expected handler for %s, got %s",
					expectedType,
					receivedType,
				)
			}
		})
	}
}

func TestDispatcherPreservesHandlerError(t *testing.T) {
	expectedError := errors.New(
		"email provider unavailable",
	)

	defaultHandler := jobHandlerStub{}

	handlers := completeHandlerRegistry(defaultHandler)
	handlers[job.TypeSendEmail] = jobHandlerStub{
		handle: func(
			context.Context,
			*job.Job,
		) error {
			return expectedError
		},
	}

	dispatcher, err := executor.NewDispatcher(handlers)
	if err != nil {
		t.Fatalf("create dispatcher: %v", err)
	}

	entity := newDispatcherTestJob(
		t,
		job.TypeSendEmail,
	)

	err = dispatcher.Execute(
		context.Background(),
		entity,
	)

	if !errors.Is(err, expectedError) {
		t.Fatalf(
			"expected handler error, got %v",
			err,
		)
	}
}

func TestDispatcherRejectsMissingHandler(t *testing.T) {
	handlers := map[job.Type]executor.Handler{
		job.TypeSendEmail:   jobHandlerStub{},
		job.TypeDataCleanup: jobHandlerStub{},
	}

	_, err := executor.NewDispatcher(handlers)

	if !errors.Is(
		err,
		executor.ErrHandlerNotRegistered,
	) {
		t.Fatalf(
			"expected ErrHandlerNotRegistered, got %v",
			err,
		)
	}
}

func TestDispatcherRejectsNilHandler(t *testing.T) {
	handlers := completeHandlerRegistry(
		jobHandlerStub{},
	)

	handlers[job.TypeReportGeneration] = nil

	_, err := executor.NewDispatcher(handlers)

	if !errors.Is(
		err,
		executor.ErrHandlerNotRegistered,
	) {
		t.Fatalf(
			"expected ErrHandlerNotRegistered, got %v",
			err,
		)
	}
}

func TestDispatcherRejectsUnsupportedRegistration(
	t *testing.T,
) {
	handlers := completeHandlerRegistry(
		jobHandlerStub{},
	)

	handlers[job.Type("unknown")] = jobHandlerStub{}

	_, err := executor.NewDispatcher(handlers)

	if !errors.Is(err, job.ErrInvalidType) {
		t.Fatalf(
			"expected ErrInvalidType, got %v",
			err,
		)
	}
}

func TestDispatcherRejectsNilJob(t *testing.T) {
	dispatcher, err := executor.NewDispatcher(
		completeHandlerRegistry(jobHandlerStub{}),
	)
	if err != nil {
		t.Fatalf("create dispatcher: %v", err)
	}

	err = dispatcher.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error for nil job")
	}
}

func TestDispatcherCopiesHandlerRegistry(t *testing.T) {
	handlers := completeHandlerRegistry(
		jobHandlerStub{},
	)

	dispatcher, err := executor.NewDispatcher(handlers)
	if err != nil {
		t.Fatalf("create dispatcher: %v", err)
	}

	delete(handlers, job.TypeSendEmail)

	entity := newDispatcherTestJob(
		t,
		job.TypeSendEmail,
	)

	if err := dispatcher.Execute(
		context.Background(),
		entity,
	); err != nil {
		t.Fatalf(
			"dispatcher registry changed externally: %v",
			err,
		)
	}
}
