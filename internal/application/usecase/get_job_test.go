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

func TestGetJobReturnsPersistedJob(t *testing.T) {
	expected, err := job.New(job.NewParams{
		ID:        job.ID(useCaseTestJobID),
		Type:      job.TypeReportGeneration,
		Payload:   json.RawMessage(`{"report":"monthly"}`),
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create expected job: %v", err)
	}

	var receivedID job.ID
	repository := repositoryStub{
		getByID: func(_ context.Context, id job.ID) (*job.Job, error) {
			receivedID = id
			return expected, nil
		},
	}

	getJob, err := usecase.NewGetJob(repository)
	if err != nil {
		t.Fatalf("NewGetJob() returned an unexpected error: %v", err)
	}

	result, err := getJob.Execute(context.Background(), usecase.GetJobInput{
		ID: "  " + useCaseTestJobID + "  ",
	})
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	if receivedID != job.ID(useCaseTestJobID) {
		t.Errorf("expected repository ID %q, got %q", useCaseTestJobID, receivedID)
	}

	if result != expected {
		t.Fatal("expected the repository job to be returned unchanged")
	}
}

func TestGetJobRejectsInvalidIDBeforeRepositoryCall(t *testing.T) {
	repositoryCalled := false
	repository := repositoryStub{
		getByID: func(context.Context, job.ID) (*job.Job, error) {
			repositoryCalled = true
			return nil, nil
		},
	}

	getJob, err := usecase.NewGetJob(repository)
	if err != nil {
		t.Fatalf("NewGetJob() returned an unexpected error: %v", err)
	}

	_, err = getJob.Execute(context.Background(), usecase.GetJobInput{
		ID: "not-a-uuid",
	})
	if !errors.Is(err, job.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	if repositoryCalled {
		t.Fatal("repository should not be called for an invalid ID")
	}
}

func TestGetJobPreservesNotFoundError(t *testing.T) {
	repository := repositoryStub{
		getByID: func(context.Context, job.ID) (*job.Job, error) {
			return nil, ports.ErrJobNotFound
		},
	}

	getJob, err := usecase.NewGetJob(repository)
	if err != nil {
		t.Fatalf("NewGetJob() returned an unexpected error: %v", err)
	}

	_, err = getJob.Execute(context.Background(), usecase.GetJobInput{
		ID: useCaseTestJobID,
	})
	if !errors.Is(err, ports.ErrJobNotFound) {
		t.Fatalf("expected ErrJobNotFound, got %v", err)
	}
}

func TestGetJobPreservesRepositoryError(t *testing.T) {
	expectedError := errors.New("database unavailable")
	repository := repositoryStub{
		getByID: func(context.Context, job.ID) (*job.Job, error) {
			return nil, expectedError
		},
	}

	getJob, err := usecase.NewGetJob(repository)
	if err != nil {
		t.Fatalf("NewGetJob() returned an unexpected error: %v", err)
	}

	_, err = getJob.Execute(context.Background(), usecase.GetJobInput{
		ID: useCaseTestJobID,
	})
	if !errors.Is(err, expectedError) {
		t.Fatalf("expected repository error to be preserved, got %v", err)
	}
}

func TestNewGetJobRejectsNilRepository(t *testing.T) {
	_, err := usecase.NewGetJob(nil)
	if err == nil {
		t.Fatal("expected an error for a nil repository")
	}
}
