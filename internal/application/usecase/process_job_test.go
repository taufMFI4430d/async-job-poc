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

func assertErrorIs(
	t *testing.T,
	err, target error,
	message string,
	args ...any,
) {
	t.Helper()

	if !errors.Is(err, target) {
		t.Fatalf(message, args...)
	}
}

func assertProcessJobAttempt(
	t *testing.T,
	attempt, totalAttempts int,
	err, executionError error,
) {
	t.Helper()

	assertErrorIs(
		t,
		err,
		executionError,
		"expected execution error, got %v",
		err,
	)

	if attempt < totalAttempts {
		assertErrorIs(
			t,
			err,
			usecase.ErrJobRetryScheduled,
			"attempt %d: expected retry scheduled, got %v",
			attempt,
			err,
		)
		return
	}

	assertErrorIs(
		t,
		err,
		usecase.ErrJobRetriesExhausted,
		"expected retries exhausted, got %v",
		err,
	)
}

func assertQueueContainsOnlyID(
	t *testing.T,
	jobIDs []job.ID,
	expectedID job.ID,
) {
	t.Helper()

	if len(jobIDs) != job.DefaultMaxRetries {
		t.Fatalf(
			"expected %d requeues, got %d",
			job.DefaultMaxRetries,
			len(jobIDs),
		)
	}

	for _, queuedID := range jobIDs {
		if queuedID != expectedID {
			t.Fatalf(
				"expected queued ID %s, got %s",
				expectedID,
				queuedID,
			)
		}
	}
}

func assertStatuses(
	t *testing.T,
	persistedStatuses, expectedStatuses []job.Status,
) {
	t.Helper()

	if len(persistedStatuses) != len(expectedStatuses) {
		t.Fatalf(
			"expected %d status updates, got %d",
			len(expectedStatuses),
			len(persistedStatuses),
		)
	}

	for index, expectedStatus := range expectedStatuses {
		if persistedStatuses[index] != expectedStatus {
			t.Fatalf(
				"update %d: expected %s, got %s",
				index,
				expectedStatus,
				persistedStatuses[index],
			)
		}
	}
}

type retryQueueStub struct {
	enqueue func(context.Context, job.ID) error
	jobIDs  []job.ID
}

func (queue *retryQueueStub) Enqueue(
	ctx context.Context,
	jobID job.ID,
) error {
	queue.jobIDs = append(queue.jobIDs, jobID)

	if queue.enqueue == nil {
		return nil
	}

	return queue.enqueue(ctx, jobID)
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
		&jobQueueStub{},
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
		&jobQueueStub{},
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

func TestNewProcessJobRejectsNilRepository(t *testing.T) {
	_, err := usecase.NewProcessJob(
		nil,
		&jobQueueStub{},
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
		&jobQueueStub{},
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
		&jobQueueStub{},
		&jobExecutorStub{},
		nil,
	)

	if err == nil {
		t.Fatal("expected an error for nil clock")
	}
}

func TestProcessJobRetriesThreeTimesThenFails(
	t *testing.T,
) {
	fixedTime := time.Date(
		2026,
		time.August,
		10,
		13,
		0,
		0,
		0,
		time.UTC,
	)

	entity := newProcessJobTestEntity(t, fixedTime)
	executionError := errors.New(
		"email provider unavailable",
	)

	var persistedStatuses []job.Status
	var persistedRetryCounts []int

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

			persistedRetryCounts = append(
				persistedRetryCounts,
				updated.RetryCount(),
			)

			return nil
		},
	}

	queue := &retryQueueStub{}

	executionCount := 0

	executor := &jobExecutorStub{
		execute: func(
			context.Context,
			*job.Job,
		) error {
			executionCount++
			return executionError
		},
	}

	processJob, err := usecase.NewProcessJob(
		repository,
		queue,
		executor,
		clockStub{now: fixedTime},
	)
	if err != nil {
		t.Fatalf("create ProcessJob use case: %v", err)
	}

	totalAttempts := job.DefaultMaxRetries + 1

	for attempt := 1; attempt <= totalAttempts; attempt++ {
		err := processJob.Execute(
			context.Background(),
			entity.ID(),
		)

		if !errors.Is(err, executionError) {
			t.Fatalf(
				"attempt %d: expected execution error, got %v",
				attempt,
				err,
			)
		}

		if attempt <= job.DefaultMaxRetries {
			if !errors.Is(
				err,
				usecase.ErrJobRetryScheduled,
			) {
				t.Fatalf(
					"attempt %d: expected retry scheduled, got %v",
					attempt,
					err,
				)
			}

			continue
		}

		if !errors.Is(
			err,
			usecase.ErrJobRetriesExhausted,
		) {
			t.Fatalf(
				"expected retries exhausted, got %v",
				err,
			)
		}
	}

	if executionCount != totalAttempts {
		t.Fatalf(
			"expected %d executions, got %d",
			totalAttempts,
			executionCount,
		)
	}

	if len(queue.jobIDs) != job.DefaultMaxRetries {
		t.Fatalf(
			"expected %d requeues, got %d",
			job.DefaultMaxRetries,
			len(queue.jobIDs),
		)
	}

	for _, queuedID := range queue.jobIDs {
		if queuedID != entity.ID() {
			t.Fatalf(
				"expected queued ID %s, got %s",
				entity.ID(),
				queuedID,
			)
		}
	}

	expectedStatuses := []job.Status{
		job.StatusProcessing,
		job.StatusPending,

		job.StatusProcessing,
		job.StatusPending,

		job.StatusProcessing,
		job.StatusPending,

		job.StatusProcessing,
		job.StatusFailed,
	}

	if len(persistedStatuses) != len(expectedStatuses) {
		t.Fatalf(
			"expected %d status updates, got %d",
			len(expectedStatuses),
			len(persistedStatuses),
		)
	}

	for index, expectedStatus := range expectedStatuses {
		if persistedStatuses[index] != expectedStatus {
			t.Fatalf(
				"update %d: expected %s, got %s",
				index,
				expectedStatus,
				persistedStatuses[index],
			)
		}
	}

	if entity.Status() != job.StatusFailed {
		t.Fatalf(
			"expected final status failed, got %s",
			entity.Status(),
		)
	}

	if entity.RetryCount() != job.DefaultMaxRetries {
		t.Fatalf(
			"expected retry count %d, got %d",
			job.DefaultMaxRetries,
			entity.RetryCount(),
		)
	}

	if _, exists := entity.CompletedAt(); !exists {
		t.Fatal(
			"terminally failed job must have completed_at",
		)
	}

	lastError, exists := entity.LastError()
	if !exists {
		t.Fatal("expected final failure reason")
	}

	if lastError != executionError.Error() {
		t.Fatalf(
			"expected last error %q, got %q",
			executionError,
			lastError,
		)
	}
}

func TestProcessJobPreservesRetryQueueFailure(
	t *testing.T,
) {
	fixedTime := time.Now().UTC()
	entity := newProcessJobTestEntity(t, fixedTime)

	executionError := errors.New(
		"email provider unavailable",
	)
	queueError := errors.New("Redis unavailable")

	repository := repositoryStub{
		getByID: func(
			context.Context,
			job.ID,
		) (*job.Job, error) {
			return entity, nil
		},
	}

	queue := &retryQueueStub{
		enqueue: func(
			context.Context,
			job.ID,
		) error {
			return queueError
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
		queue,
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

	if !errors.Is(err, queueError) {
		t.Fatalf(
			"expected queue error, got %v",
			err,
		)
	}

	if entity.Status() != job.StatusPending {
		t.Fatalf(
			"expected pending status, got %s",
			entity.Status(),
		)
	}

	if entity.RetryCount() != 1 {
		t.Fatalf(
			"expected retry count 1, got %d",
			entity.RetryCount(),
		)
	}
}

func TestNewProcessJobRejectsNilQueue(t *testing.T) {
	_, err := usecase.NewProcessJob(
		repositoryStub{},
		nil,
		&jobExecutorStub{},
		clockStub{now: time.Now()},
	)

	if err == nil {
		t.Fatal("expected an error for nil queue")
	}
}
