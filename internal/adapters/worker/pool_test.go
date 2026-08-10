package worker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/adapters/worker"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

var poolTestJobIDs = []job.ID{
	"4e2cebf4-7a09-4750-942d-d75c74df9917",
	"705ab754-ac2d-465f-af38-cf10a3ccf2ca",
	"414b2f53-e52c-48bd-b405-bd8a6bfa61cc",
	"c5942e44-73b3-4563-97a5-a20c993982cc",
	"b06c1314-5b3c-4089-ae72-e1890ef93f49",
}

func TestPoolProcessesFiveJobsConcurrently(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nextJobIndex := 0

	consumer := &dispatcherConsumerStub{
		dequeue: func(ctx context.Context) (job.ID, error) {
			if nextJobIndex < len(poolTestJobIDs) {
				jobID := poolTestJobIDs[nextJobIndex]
				nextJobIndex++
				return jobID, nil
			}

			<-ctx.Done()
			return "", ctx.Err()
		},
	}

	var mutex sync.Mutex

	currentlyProcessing := 0
	maximumConcurrent := 0
	processedJobs := 0

	allWorkersStarted := make(chan struct{})
	allJobsProcessed := make(chan struct{})
	releaseWorkers := make(chan struct{})

	processor := &processorWorkerStub{
		execute: func(
			_ context.Context,
			_ job.ID,
		) error {
			mutex.Lock()

			currentlyProcessing++

			if currentlyProcessing > maximumConcurrent {
				maximumConcurrent = currentlyProcessing
			}

			if currentlyProcessing == worker.WorkerCount {
				close(allWorkersStarted)
			}

			mutex.Unlock()

			// Keep every worker busy until all five have started.
			<-releaseWorkers

			mutex.Lock()

			currentlyProcessing--
			processedJobs++

			if processedJobs == len(poolTestJobIDs) {
				close(allJobsProcessed)
			}

			mutex.Unlock()

			return nil
		},
	}

	workerPool, err := worker.NewPool(
		consumer,
		processor,
		newDispatcherTestLogger(),
	)
	if err != nil {
		t.Fatalf("create worker pool: %v", err)
	}

	runResult := make(chan error, 1)

	go func() {
		runResult <- workerPool.Run(ctx)
	}()

	select {
	case <-allWorkersStarted:
	case <-time.After(2 * time.Second):
		t.Fatal(
			"five workers did not start processing concurrently",
		)
	}

	close(releaseWorkers)

	select {
	case <-allJobsProcessed:
	case <-time.After(2 * time.Second):
		t.Fatal("jobs did not finish processing")
	}

	cancel()

	select {
	case err := <-runResult:
		if err != nil {
			t.Fatalf("run worker pool: %v", err)
		}

	case <-time.After(2 * time.Second):
		t.Fatal(
			"worker pool did not stop after cancellation",
		)
	}

	mutex.Lock()
	defer mutex.Unlock()

	if maximumConcurrent != worker.WorkerCount {
		t.Fatalf(
			"expected %d concurrent workers, got %d",
			worker.WorkerCount,
			maximumConcurrent,
		)
	}

	if processedJobs != len(poolTestJobIDs) {
		t.Fatalf(
			"expected %d processed jobs, got %d",
			len(poolTestJobIDs),
			processedJobs,
		)
	}
}

func TestNewPoolRejectsNilConsumer(t *testing.T) {
	processor := &processorWorkerStub{
		execute: func(
			context.Context,
			job.ID,
		) error {
			return nil
		},
	}

	_, err := worker.NewPool(
		nil,
		processor,
		newDispatcherTestLogger(),
	)
	if err == nil {
		t.Fatal("expected an error for nil consumer")
	}
}

func TestNewPoolRejectsNilProcessor(t *testing.T) {
	consumer := &dispatcherConsumerStub{
		dequeue: func(
			context.Context,
		) (job.ID, error) {
			return "", nil
		},
	}

	_, err := worker.NewPool(
		consumer,
		nil,
		newDispatcherTestLogger(),
	)
	if err == nil {
		t.Fatal("expected an error for nil processor")
	}
}

func TestNewPoolRejectsNilLogger(t *testing.T) {
	consumer := &dispatcherConsumerStub{
		dequeue: func(
			context.Context,
		) (job.ID, error) {
			return "", nil
		},
	}

	processor := &processorWorkerStub{
		execute: func(
			context.Context,
			job.ID,
		) error {
			return nil
		},
	}

	_, err := worker.NewPool(
		consumer,
		processor,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for nil logger")
	}
}
