package redisadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

type JobQueue struct {
	client    *redis.Client
	queueName string
}

var (
	_ ports.JobQueue    = (*JobQueue)(nil)
	_ ports.JobConsumer = (*JobQueue)(nil)
)

func NewJobQueue(
	client *redis.Client,
	queueName string,
) (*JobQueue, error) {
	if client == nil {
		return nil, errors.New("Redis client must not be nil")
	}

	normalizedQueueName := strings.TrimSpace(queueName)
	if normalizedQueueName == "" {
		return nil, errors.New("Redis queue name must not be empty")
	}

	return &JobQueue{
		client:    client,
		queueName: normalizedQueueName,
	}, nil
}

func (queue *JobQueue) Enqueue(
	ctx context.Context,
	id job.ID,
) error {
	if !id.IsValid() {
		return fmt.Errorf("%w: %q", job.ErrInvalidID, id)
	}

	if err := queue.client.LPush(
		ctx,
		queue.queueName,
		id.String(),
	).Err(); err != nil {
		return fmt.Errorf(
			"%w: enqueue job %s: %w",
			ports.ErrJobQueueUnavailable,
			id,
			err,
		)
	}

	return nil
}

func (queue *JobQueue) Dequeue(
	ctx context.Context,
) (job.ID, error) {
	result, err := queue.client.BRPop(
		ctx,
		0,
		queue.queueName,
	).Result()
	if err != nil {
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}

		return "", fmt.Errorf(
			"%w: dequeue from %s: %w",
			ports.ErrJobQueueUnavailable,
			queue.queueName,
			err,
		)
	}

	if len(result) != 2 {
		return "", fmt.Errorf(
			"%w: unexpected Redis response length %d",
			ports.ErrJobQueueUnavailable,
			len(result),
		)
	}

	jobID, err := job.ParseID(result[1])
	if err != nil {
		return "", fmt.Errorf(
			"invalid job ID received from queue %s: %w",
			queue.queueName,
			err,
		)
	}

	return jobID, nil
}
