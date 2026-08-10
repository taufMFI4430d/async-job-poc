package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/adapters/worker"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const secondProcessorWorkerTestJobID = "50e742c5-6f92-42f5-8ea2-b086d09c1637"

type processorWorkerStub struct {
	execute func(context.Context, job.ID) error
}

func (processor *processorWorkerStub) Execute(
	ctx context.Context,
	jobID job.ID,
) error {
	return processor.execute(ctx, jobID)
}

func TestProcessorWorkerProcessesJobsUntilChannelCloses(
	t *testing.T,
) {
	jobIDs := []job.ID{
		job.ID(dispatcherTestJobID),
		job.ID(secondProcessorWorkerTestJobID),
	}

	jobs := make(chan job.ID, len(jobIDs))

	for _, jobID := range jobIDs {
		jobs <- jobID
	}

	close(jobs)

	var processedIDs []job.ID

	processor := &processorWorkerStub{
		execute: func(
			_ context.Context,
			jobID job.ID,
		) error {
			processedIDs = append(processedIDs, jobID)
			return nil
		},
	}

	processingWorker, err := worker.NewProcessorWorker(
		1,
		processor,
		newDispatcherTestLogger(),
	)
	if err != nil {
		t.Fatalf("create processing worker: %v", err)
	}

	if err := processingWorker.Run(
		context.Background(),
		jobs,
	); err != nil {
		t.Fatalf("run processing worker: %v", err)
	}

	if len(processedIDs) != len(jobIDs) {
		t.Fatalf(
			"expected %d processed jobs, got %d",
			len(jobIDs),
			len(processedIDs),
		)
	}

	for index, expectedID := range jobIDs {
		if processedIDs[index] != expectedID {
			t.Fatalf(
				"expected job %d to be %s, got %s",
				index,
				expectedID,
				processedIDs[index],
			)
		}
	}
}

func TestProcessorWorkerContinuesAfterJobFailure(
	t *testing.T,
) {
	firstID := job.ID(dispatcherTestJobID)
	secondID := job.ID(secondProcessorWorkerTestJobID)

	jobs := make(chan job.ID, 2)
	jobs <- firstID
	jobs <- secondID
	close(jobs)

	executionError := errors.New("simulated job failure")
	var processedIDs []job.ID

	processor := &processorWorkerStub{
		execute: func(
			_ context.Context,
			jobID job.ID,
		) error {
			processedIDs = append(processedIDs, jobID)

			if jobID == firstID {
				return executionError
			}

			return nil
		},
	}

	processingWorker, err := worker.NewProcessorWorker(
		2,
		processor,
		newDispatcherTestLogger(),
	)
	if err != nil {
		t.Fatalf("create processing worker: %v", err)
	}

	if err := processingWorker.Run(
		context.Background(),
		jobs,
	); err != nil {
		t.Fatalf("run processing worker: %v", err)
	}

	if len(processedIDs) != 2 {
		t.Fatalf(
			"expected worker to process 2 jobs, got %d",
			len(processedIDs),
		)
	}

	if processedIDs[1] != secondID {
		t.Fatalf(
			"worker did not continue to the second job",
		)
	}
}

func TestProcessorWorkerStopsWhenContextIsCancelled(
	t *testing.T,
) {
	ctx, cancel := context.WithCancel(context.Background())

	jobs := make(chan job.ID)
	processorCalled := false

	processor := &processorWorkerStub{
		execute: func(
			context.Context,
			job.ID,
		) error {
			processorCalled = true
			return nil
		},
	}

	processingWorker, err := worker.NewProcessorWorker(
		3,
		processor,
		newDispatcherTestLogger(),
	)
	if err != nil {
		t.Fatalf("create processing worker: %v", err)
	}

	runResult := make(chan error, 1)

	go func() {
		runResult <- processingWorker.Run(ctx, jobs)
	}()

	cancel()

	select {
	case err := <-runResult:
		if err != nil {
			t.Fatalf("run processing worker: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"processing worker did not stop after cancellation",
		)
	}

	if processorCalled {
		t.Fatal(
			"processor must not run when no job was received",
		)
	}
}

func TestProcessorWorkerRejectsNilJobsChannel(
	t *testing.T,
) {
	processor := &processorWorkerStub{
		execute: func(
			context.Context,
			job.ID,
		) error {
			return nil
		},
	}

	processingWorker, err := worker.NewProcessorWorker(
		1,
		processor,
		newDispatcherTestLogger(),
	)
	if err != nil {
		t.Fatalf("create processing worker: %v", err)
	}

	err = processingWorker.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error for nil jobs channel")
	}
}

func TestNewProcessorWorkerRejectsInvalidID(
	t *testing.T,
) {
	processor := &processorWorkerStub{
		execute: func(
			context.Context,
			job.ID,
		) error {
			return nil
		},
	}

	_, err := worker.NewProcessorWorker(
		0,
		processor,
		newDispatcherTestLogger(),
	)
	if err == nil {
		t.Fatal("expected an error for invalid worker ID")
	}
}

func TestNewProcessorWorkerRejectsNilProcessor(
	t *testing.T,
) {
	_, err := worker.NewProcessorWorker(
		1,
		nil,
		newDispatcherTestLogger(),
	)
	if err == nil {
		t.Fatal("expected an error for nil processor")
	}
}

func TestNewProcessorWorkerRejectsNilLogger(
	t *testing.T,
) {
	processor := &processorWorkerStub{
		execute: func(
			context.Context,
			job.ID,
		) error {
			return nil
		},
	}

	_, err := worker.NewProcessorWorker(
		1,
		processor,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for nil logger")
	}
}
