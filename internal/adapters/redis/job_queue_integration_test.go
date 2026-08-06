//go:build integration

package redisadapter_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	redisadapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/redis"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/cache"
)

func TestJobQueueAgainstRedis(t *testing.T) {
	requiredEnvironment := []string{
		"REDIS_HOST",
		"REDIS_PORT",
	}

	for _, name := range requiredEnvironment {
		if os.Getenv(name) == "" {
			t.Skipf(
				"%s is required for the Redis integration test",
				name,
			)
		}
	}

	redisPort, err := strconv.Atoi(os.Getenv("REDIS_PORT"))
	if err != nil {
		t.Fatalf("parse REDIS_PORT: %v", err)
	}

	redisDB := 0
	if rawDB := os.Getenv("REDIS_DB"); rawDB != "" {
		redisDB, err = strconv.Atoi(rawDB)
		if err != nil {
			t.Fatalf("parse REDIS_DB: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	connection, err := cache.OpenRedis(
		ctx,
		cache.RedisOptions{
			Host:     os.Getenv("REDIS_HOST"),
			Port:     redisPort,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
	)
	if err != nil {
		t.Fatalf("open Redis: %v", err)
	}

	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close Redis: %v", err)
		}
	})

	queueName := fmt.Sprintf(
		"test:jobs:pending:%d",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		if err := connection.Client().
			Del(context.Background(), queueName).
			Err(); err != nil {
			t.Errorf("delete test queue: %v", err)
		}
	})

	queue, err := redisadapter.NewJobQueue(
		connection.Client(),
		queueName,
	)
	if err != nil {
		t.Fatalf("create job queue: %v", err)
	}

	expectedID := job.ID(
		"84cd35f4-ddc6-46ca-986f-99de8b497280",
	)

	if err := queue.Enqueue(ctx, expectedID); err != nil {
		t.Fatalf("enqueue job: %v", err)
	}

	queuedID, err := queue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue job: %v", err)
	}

	if queuedID != expectedID {
		t.Fatalf(
			"expected queued ID %q, got %q",
			expectedID,
			queuedID,
		)
	}
}

func TestJobQueueDequeueStopsWhenContextIsCancelled(t *testing.T) {
	requiredEnvironment := []string{
		"REDIS_HOST",
		"REDIS_PORT",
	}

	for _, name := range requiredEnvironment {
		if os.Getenv(name) == "" {
			t.Skipf(
				"%s is required for the Redis integration test",
				name,
			)
		}
	}

	redisPort, err := strconv.Atoi(os.Getenv("REDIS_PORT"))
	if err != nil {
		t.Fatalf("parse REDIS_PORT: %v", err)
	}

	redisDB := 0
	if rawDB := os.Getenv("REDIS_DB"); rawDB != "" {
		redisDB, err = strconv.Atoi(rawDB)
		if err != nil {
			t.Fatalf("parse REDIS_DB: %v", err)
		}
	}

	connection, err := cache.OpenRedis(
		context.Background(),
		cache.RedisOptions{
			Host:     os.Getenv("REDIS_HOST"),
			Port:     redisPort,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
	)
	if err != nil {
		t.Fatalf("open Redis: %v", err)
	}

	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close Redis: %v", err)
		}
	})

	queueName := fmt.Sprintf(
		"test:jobs:pending:cancellation:%d",
		time.Now().UnixNano(),
	)

	queue, err := redisadapter.NewJobQueue(
		connection.Client(),
		queueName,
	)
	if err != nil {
		t.Fatalf("create job queue: %v", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		100*time.Millisecond,
	)
	defer cancel()

	_, err = queue.Dequeue(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf(
			"expected context deadline exceeded, got %v",
			err,
		)
	}
}
