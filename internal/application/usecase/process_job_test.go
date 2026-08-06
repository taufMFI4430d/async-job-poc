package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/application/usecase"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

type jobExecutorStub struct {
	execute func(context.Context, *job.Job) error
	called  bool
}

func (executor *jobExecutorStub) Execute(
	ctx context.Context,
	entity *job.Job,
) error {
	executor.called = true

	if executor.execute == nil {
		return nil
	}

	return executor.execute(ctx, entity)
}

func newProcessJobTestEntity(
	t *testing.T,
	createdAt time.Time,
) *job.Job {
	t.Helper()

	entity, err := job.New(job.NewParams{
		ID:        job.ID(useCaseTestJobID),
		Type:      job.TypeSendEmail,
		Payload:   json.RawMessage(`{"to":"learner@example.com"}`),
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("create test job: %v", err)
	}

	return entity
}

func TestProcessJobPersistsProcessingAndSuccess(t *testing.T) {
	fixedTime := time.Date(
		2026,
		time.August,
		5,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	entity := newProcessJobTestEntity(t, fixedTime)

	var persistedStatuses []job.Status

	repository := repositoryStub{
		getByID: func(
			context.Context,
			job.ID,
		) (*job.Job, error) {
			return entity, nil
		},
		update: func(
			_ context.Context,
			updated *job.Job,
		) error {
			persistedStatuses = append(
				persistedStatuses,
				updated.Status(),
			)
			return nil
		},
	}

	executor := &jobExecutorStub{
		execute: func(
			_ context.Context,
			received *job.Job,
		) error {
			if received.Status() != job.StatusProcessing {
				t.Fatalf(
					"expected executor to receive processing job, got %s",
					received.Status(),
				)
			}

			return nil
		},
	}

	processJob, err := usecase.NewProcessJob(
		repository,
		executor,
		clockStub{now: fixedTime},
	)
	if err != nil {
		t.Fatalf("create ProcessJob use case: %v", err)
	}

	err = processJob.Execute(
		context.Background(),
		entity.ID(),
	)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}

	if !executor.called {
		t.Fatal("expected executor to be called")
	}

	expectedStatuses := []job.Status{
		job.StatusProcessing,
		job.StatusSuccess,
	}

	if len(persistedStatuses) != len(expectedStatuses) {
		t.Fatalf(
			"expected %d updates, got %d",
			len(expectedStatuses),
			len(persistedStatuses),
		)
	}

	for index, expected := range expectedStatuses {
		if persistedStatuses[index] != expected {
			t.Fatalf(
				"expected update %d to have status %s, got %s",
				index,
				expected,
				persistedStatuses[index],
			)
		}
	}

	if entity.Status() != job.StatusSuccess {
		t.Fatalf(
			"expected final status success, got %s",
			entity.Status(),
		)
	}
}

func TestProcessJobPersistsFailedStatus(t *testing.T) {
	fixedTime := time.Date(
		2026,
		time.August,
		5,
		13,
		0,
		0,
		0,
		time.UTC,
	)

	entity := newProcessJobTestEntity(t, fixedTime)
	executionError := errors.New("email provider unavailable")

	var persistedStatuses []job.Status

	repository := repositoryStub{
		getByID: func(
			context.Context,
			job.ID,
		) (*job.Job, error) {
			return entity, nil
		},
		update: func(
			_ context.Context,
			updated *job.Job,
		) error {
			persistedStatuses = append(
				persistedStatuses,
				updated.Status(),
			)
			return nil
		},
	}

	executor := &jobExecutorStub{
		execute: func(
			context.Context,
			*job.Job,
		) error {
			return executionError
		},
	}

	processJob, err := usecase.NewProcessJob(
		repository,
		executor,
		clockStub{now: fixedTime},
	)
	if err != nil {
		t.Fatalf("create ProcessJob use case: %v", err)
	}

	err = processJob.Execute(
		context.Background(),
		entity.ID(),
	)

	if !errors.Is(err, executionError) {
		t.Fatalf(
			"expected execution error, got %v",
			err,
		)
	}

	expectedStatuses := []job.Status{
		job.StatusProcessing,
		job.StatusFailed,
	}

	if len(persistedStatuses) != len(expectedStatuses) {
		t.Fatalf(
			"expected %d updates, got %d",
			len(expectedStatuses),
			len(persistedStatuses),
		)
	}

	for index, expected := range expectedStatuses {
		if persistedStatuses[index] != expected {
			t.Fatalf(
				"expected update %d to have status %s, got %s",
				index,
				expected,
				persistedStatuses[index],
			)
		}
	}

	lastError, exists := entity.LastError()
	if !exists {
		t.Fatal("expected failed job to have last_error")
	}

	if lastError != executionError.Error() {
		t.Fatalf(
			"expected last error %q, got %q",
			executionError,
			lastError,
		)
	}
}

func TestProcessJobDoesNotExecuteWhenProcessingStatusCannotBePersisted(
	t *testing.T,
) {
	fixedTime := time.Now().UTC()
	entity := newProcessJobTestEntity(t, fixedTime)

	updateError := errors.New("database unavailable")

	repository := repositoryStub{
		getByID: func(
			context.Context,
			job.ID,
		) (*job.Job, error) {
			return entity, nil
		},
		update: func(
			context.Context,
			*job.Job,
		) error {
			return updateError
		},
	}

	executor := &jobExecutorStub{}

	processJob, err := usecase.NewProcessJob(
		repository,
		executor,
		clockStub{now: fixedTime},
	)
	if err != nil {
		t.Fatalf("create ProcessJob use case: %v", err)
	}

	err = processJob.Execute(
		context.Background(),
		entity.ID(),
	)

	if !errors.Is(err, updateError) {
		t.Fatalf(
			"expected repository update error, got %v",
			err,
		)
	}

	if executor.called {
		t.Fatal(
			"executor must not run when processing status was not persisted",
		)
	}
}

func TestProcessJobRejectsInvalidID(t *testing.T) {
	processJob, err := usecase.NewProcessJob(
		repositoryStub{},
		&jobExecutorStub{},
		clockStub{now: time.Now()},
	)
	if err != nil {
		t.Fatalf("create ProcessJob use case: %v", err)
	}

	err = processJob.Execute(
		context.Background(),
		job.ID("invalid"),
	)

	if !errors.Is(err, job.ErrInvalidID) {
		t.Fatalf(
			"expected ErrInvalidID, got %v",
			err,
		)
	}
}

func TestNewProcessJobRejectsNilRepository(t *testing.T) {
	_, err := usecase.NewProcessJob(
		nil,
		&jobExecutorStub{},
		clockStub{now: time.Now()},
	)
	if err == nil {
		t.Fatal("expected an error for nil repository")
	}
}

func TestNewProcessJobRejectsNilExecutor(t *testing.T) {
	_, err := usecase.NewProcessJob(
		repositoryStub{},
		nil,
		clockStub{now: time.Now()},
	)
	if err == nil {
		t.Fatal("expected an error for nil executor")
	}
}

func TestNewProcessJobRejectsNilClock(t *testing.T) {
	_, err := usecase.NewProcessJob(
		repositoryStub{},
		&jobExecutorStub{},
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for nil clock")
	}
}
