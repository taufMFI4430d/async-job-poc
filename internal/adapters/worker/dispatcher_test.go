package worker_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/adapters/worker"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const dispatcherTestJobID = "127362d1-7eb7-4940-b165-c202367bf172"

type dispatcherConsumerStub struct {
	dequeue func(context.Context) (job.ID, error)
}

func (consumer *dispatcherConsumerStub) Dequeue(
	ctx context.Context,
) (job.ID, error) {
	return consumer.dequeue(ctx)
}

func newDispatcherTestLogger() *slog.Logger {
	return slog.New(
		slog.NewTextHandler(io.Discard, nil),
	)
}

func TestDispatcherForwardsDequeuedJobAndClosesChannel(
	t *testing.T,
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expectedID := job.ID(dispatcherTestJobID)
	dequeueCalls := 0

	consumer := &dispatcherConsumerStub{
		dequeue: func(ctx context.Context) (job.ID, error) {
			dequeueCalls++

			if dequeueCalls == 1 {
				return expectedID, nil
			}

			<-ctx.Done()
			return "", ctx.Err()
		},
	}

	dispatcher, err := worker.NewDispatcher(
		consumer,
		newDispatcherTestLogger(),
	)
	if err != nil {
		t.Fatalf("create dispatcher: %v", err)
	}

	jobs := make(chan job.ID)
	runResult := make(chan error, 1)

	go func() {
		runResult <- dispatcher.Run(ctx, jobs)
	}()

	select {
	case receivedID := <-jobs:
		if receivedID != expectedID {
			t.Fatalf(
				"expected job ID %s, got %s",
				expectedID,
				receivedID,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("timed out waiting for dispatched job")
	}

	cancel()

	select {
	case err := <-runResult:
		if err != nil {
			t.Fatalf("run dispatcher: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal("dispatcher did not stop after cancellation")
	}

	_, channelOpen := <-jobs
	if channelOpen {
		t.Fatal(
			"expected dispatcher to close the jobs channel",
		)
	}
}

func TestDispatcherStopsWhileChannelSendIsBlocked(
	t *testing.T,
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dequeued := make(chan struct{})

	consumer := &dispatcherConsumerStub{
		dequeue: func(context.Context) (job.ID, error) {
			close(dequeued)
			return job.ID(dispatcherTestJobID), nil
		},
	}

	dispatcher, err := worker.NewDispatcher(
		consumer,
		newDispatcherTestLogger(),
	)
	if err != nil {
		t.Fatalf("create dispatcher: %v", err)
	}

	// The unbuffered channel has no receiver, so sending will block.
	jobs := make(chan job.ID)
	runResult := make(chan error, 1)

	go func() {
		runResult <- dispatcher.Run(ctx, jobs)
	}()

	select {
	case <-dequeued:
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not dequeue a job")
	}

	cancel()

	select {
	case err := <-runResult:
		if err != nil {
			t.Fatalf("run dispatcher: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"dispatcher remained blocked during shutdown",
		)
	}
}

func TestDispatcherRejectsNilJobsChannel(t *testing.T) {
	consumer := &dispatcherConsumerStub{
		dequeue: func(
			context.Context,
		) (job.ID, error) {
			return "", nil
		},
	}

	dispatcher, err := worker.NewDispatcher(
		consumer,
		newDispatcherTestLogger(),
	)
	if err != nil {
		t.Fatalf("create dispatcher: %v", err)
	}

	err = dispatcher.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error for nil jobs channel")
	}
}

func TestNewDispatcherRejectsNilConsumer(t *testing.T) {
	_, err := worker.NewDispatcher(
		nil,
		newDispatcherTestLogger(),
	)
	if err == nil {
		t.Fatal("expected an error for nil consumer")
	}
}

func TestNewDispatcherRejectsNilLogger(t *testing.T) {
	consumer := &dispatcherConsumerStub{
		dequeue: func(
			context.Context,
		) (job.ID, error) {
			return "", nil
		},
	}

	_, err := worker.NewDispatcher(
		consumer,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for nil logger")
	}
}
