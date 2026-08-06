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

const simulatedExecutorTestJobID = "43022c37-ad99-48fb-b14f-124ec802ebcb"

func TestSimulatedExecutorCompletes(t *testing.T) {
	simulated, err := executor.NewSimulated(time.Millisecond)
	if err != nil {
		t.Fatalf("create simulated executor: %v", err)
	}

	entity, err := job.New(job.NewParams{
		ID:        job.ID(simulatedExecutorTestJobID),
		Type:      job.TypeSendEmail,
		Payload:   json.RawMessage(`{"to":"learner@example.com"}`),
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create test job: %v", err)
	}

	if err := simulated.Execute(
		context.Background(),
		entity,
	); err != nil {
		t.Fatalf("execute simulated job: %v", err)
	}
}

func TestSimulatedExecutorStopsWhenContextIsCancelled(
	t *testing.T,
) {
	simulated, err := executor.NewSimulated(time.Hour)
	if err != nil {
		t.Fatalf("create simulated executor: %v", err)
	}

	entity, err := job.New(job.NewParams{
		ID:        job.ID(simulatedExecutorTestJobID),
		Type:      job.TypeDataCleanup,
		Payload:   json.RawMessage(`{"scope":"expired"}`),
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create test job: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = simulated.Execute(ctx, entity)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"expected context cancellation, got %v",
			err,
		)
	}
}

func TestSimulatedExecutorRejectsNilJob(t *testing.T) {
	simulated, err := executor.NewSimulated(time.Millisecond)
	if err != nil {
		t.Fatalf("create simulated executor: %v", err)
	}

	err = simulated.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error for nil job")
	}
}

func TestNewSimulatedExecutorRejectsInvalidDuration(
	t *testing.T,
) {
	_, err := executor.NewSimulated(0)
	if err == nil {
		t.Fatal("expected an error for zero duration")
	}
}
