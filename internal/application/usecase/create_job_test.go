package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/application/usecase"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const useCaseTestJobID = "d3b75d2a-7485-42f3-bbf9-97c2400f897f"

type repositoryStub struct {
	create  func(context.Context, *job.Job) error
	getByID func(context.Context, job.ID) (*job.Job, error)
}

func (repository repositoryStub) Create(
	ctx context.Context,
	entity *job.Job,
) error {
	if repository.create == nil {
		return nil
	}

	return repository.create(ctx, entity)
}

func (repository repositoryStub) GetByID(
	ctx context.Context,
	id job.ID,
) (*job.Job, error) {
	if repository.getByID != nil {
		return repository.getByID(ctx, id)
	}

	return nil, ports.ErrJobNotFound
}

type idGeneratorStub struct {
	id     job.ID
	err    error
	called bool
}

func (generator *idGeneratorStub) NewID() (job.ID, error) {
	generator.called = true
	return generator.id, generator.err
}

type clockStub struct {
	now time.Time
}

func (clock clockStub) Now() time.Time {
	return clock.now
}

func TestCreateJobPersistsPendingJob(t *testing.T) {
	fixedTime := time.Date(2026, time.August, 3, 14, 30, 0, 0, time.UTC)
	idGenerator := &idGeneratorStub{id: job.ID(useCaseTestJobID)}

	var persisted *job.Job
	repository := repositoryStub{
		create: func(_ context.Context, entity *job.Job) error {
			persisted = entity
			return nil
		},
	}

	createJob, err := usecase.NewCreateJob(
		repository,
		idGenerator,
		clockStub{now: fixedTime},
	)
	if err != nil {
		t.Fatalf("NewCreateJob() returned an unexpected error: %v", err)
	}

	created, err := createJob.Execute(context.Background(), usecase.CreateJobInput{
		Type:    " SEND_EMAIL ",
		Payload: json.RawMessage(`{"to":"learner@example.com"}`),
	})
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	if persisted != created {
		t.Fatal("expected the returned job to be the persisted entity")
	}

	if created.ID() != job.ID(useCaseTestJobID) {
		t.Errorf("expected ID %q, got %q", useCaseTestJobID, created.ID())
	}

	if created.Type() != job.TypeSendEmail {
		t.Errorf("expected type %q, got %q", job.TypeSendEmail, created.Type())
	}

	if created.Status() != job.StatusPending {
		t.Errorf("expected pending status, got %q", created.Status())
	}

	if !created.CreatedAt().Equal(fixedTime) {
		t.Errorf("expected creation time %v, got %v", fixedTime, created.CreatedAt())
	}

	if created.RetryCount() != 0 || created.MaxRetries() != job.DefaultMaxRetries {
		t.Errorf(
			"unexpected retry defaults: count=%d max=%d",
			created.RetryCount(),
			created.MaxRetries(),
		)
	}
}

func TestCreateJobRejectsUnsupportedTypeBeforeGeneratingID(t *testing.T) {
	idGenerator := &idGeneratorStub{id: job.ID(useCaseTestJobID)}
	createJob, err := usecase.NewCreateJob(
		repositoryStub{},
		idGenerator,
		clockStub{now: time.Now()},
	)
	if err != nil {
		t.Fatalf("NewCreateJob() returned an unexpected error: %v", err)
	}

	_, err = createJob.Execute(context.Background(), usecase.CreateJobInput{
		Type:    "unknown",
		Payload: json.RawMessage(`{"value":"test"}`),
	})
	if !errors.Is(err, job.ErrInvalidType) {
		t.Fatalf("expected ErrInvalidType, got %v", err)
	}

	if idGenerator.called {
		t.Fatal("ID generator should not be called for an invalid job type")
	}
}

func TestCreateJobRejectsInvalidPayloadWithoutPersisting(t *testing.T) {
	persistCalled := false
	repository := repositoryStub{
		create: func(context.Context, *job.Job) error {
			persistCalled = true
			return nil
		},
	}

	createJob, err := usecase.NewCreateJob(
		repository,
		&idGeneratorStub{id: job.ID(useCaseTestJobID)},
		clockStub{now: time.Now()},
	)
	if err != nil {
		t.Fatalf("NewCreateJob() returned an unexpected error: %v", err)
	}

	_, err = createJob.Execute(context.Background(), usecase.CreateJobInput{
		Type:    job.TypeSendEmail.String(),
		Payload: json.RawMessage(`{}`),
	})
	if !errors.Is(err, job.ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}

	if persistCalled {
		t.Fatal("repository should not be called for an invalid payload")
	}
}

func TestCreateJobPreservesRepositoryError(t *testing.T) {
	expectedError := errors.New("database unavailable")
	repository := repositoryStub{
		create: func(context.Context, *job.Job) error {
			return expectedError
		},
	}

	createJob, err := usecase.NewCreateJob(
		repository,
		&idGeneratorStub{id: job.ID(useCaseTestJobID)},
		clockStub{now: time.Now()},
	)
	if err != nil {
		t.Fatalf("NewCreateJob() returned an unexpected error: %v", err)
	}

	_, err = createJob.Execute(context.Background(), usecase.CreateJobInput{
		Type:    job.TypeDataCleanup.String(),
		Payload: json.RawMessage(`{"scope":"expired"}`),
	})
	if !errors.Is(err, expectedError) {
		t.Fatalf("expected repository error to be preserved, got %v", err)
	}
}

func TestCreateJobPreservesIDGeneratorError(t *testing.T) {
	expectedError := errors.New("random source unavailable")
	createJob, err := usecase.NewCreateJob(
		repositoryStub{},
		&idGeneratorStub{err: expectedError},
		clockStub{now: time.Now()},
	)
	if err != nil {
		t.Fatalf("NewCreateJob() returned an unexpected error: %v", err)
	}

	_, err = createJob.Execute(context.Background(), usecase.CreateJobInput{
		Type:    job.TypeReportGeneration.String(),
		Payload: json.RawMessage(`{"report":"monthly"}`),
	})
	if !errors.Is(err, expectedError) {
		t.Fatalf("expected ID generator error to be preserved, got %v", err)
	}
}
