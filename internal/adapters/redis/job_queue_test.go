package redisadapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"

	redisadapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/redis"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

func TestNewJobQueueRejectsNilClient(t *testing.T) {
	_, err := redisadapter.NewJobQueue(
		nil,
		"jobs:pending",
	)

	if err == nil {
		t.Fatal("expected an error for a nil Redis client")
	}
}

func TestNewJobQueueRejectsEmptyQueueName(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	_, err := redisadapter.NewJobQueue(client, "   ")
	if err == nil {
		t.Fatal("expected an error for an empty queue name")
	}
}

func TestEnqueueRejectsInvalidJobIDWithoutCallingRedis(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1",
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	queue, err := redisadapter.NewJobQueue(
		client,
		"jobs:pending",
	)
	if err != nil {
		t.Fatalf("NewJobQueue() returned an unexpected error: %v", err)
	}

	err = queue.Enqueue(
		context.Background(),
		job.ID("invalid-id"),
	)
	if !errors.Is(err, job.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}
