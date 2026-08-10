package job

import (
	"fmt"
	"strings"
	"time"
)

type FailureOutcome string

const (
	FailureOutcomeRetryScheduled FailureOutcome = "retry_scheduled"

	FailureOutcomeTerminalFailure FailureOutcome = "terminal_failure"
)

// CanRetry reports whether another retry may be scheduled.
//
// It does not indicate whether a pending job should be processed.
// A pending job with retry_count equal to max_retries represents the
// final allowed retry attempt.
func (entity *Job) CanRetry() bool {
	return entity.retryCount < entity.maxRetries
}

// RecordFailure records the result of one unsuccessful processing attempt.
//
// When retries remain, the job returns to pending and retry_count increases.
// When retries are exhausted, the job becomes terminally failed.
func (entity *Job) RecordFailure(
	at time.Time,
	reason string,
) (FailureOutcome, error) {
	if entity.status != StatusProcessing {
		return "", fmt.Errorf(
			"%w: cannot record failure while job is %s",
			ErrInvalidTransition,
			entity.status,
		)
	}

	normalizedReason := strings.TrimSpace(reason)
	if normalizedReason == "" {
		return "", ErrInvalidFailureReason
	}

	transitionTime, err := entity.transitionTime(at)
	if err != nil {
		return "", err
	}

	entity.lastError = &normalizedReason
	entity.updatedAt = transitionTime

	if entity.CanRetry() {
		entity.retryCount++
		entity.status = StatusPending
		entity.completedAt = nil

		return FailureOutcomeRetryScheduled, nil
	}

	entity.status = StatusFailed
	entity.completedAt = &transitionTime

	return FailureOutcomeTerminalFailure, nil
}
