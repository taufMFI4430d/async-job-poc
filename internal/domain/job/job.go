package job

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const DefaultMaxRetries = 3

type NewParams struct {
	ID        ID
	Type      Type
	Payload   json.RawMessage
	CreatedAt time.Time
}

type RestoreParams struct {
	ID          ID
	Type        Type
	Status      Status
	Payload     json.RawMessage
	RetryCount  int
	MaxRetries  int
	LastError   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
}

type Job struct {
	id          ID
	jobType     Type
	status      Status
	payload     json.RawMessage
	retryCount  int
	maxRetries  int
	lastError   *string
	createdAt   time.Time
	updatedAt   time.Time
	startedAt   *time.Time
	completedAt *time.Time
}

func New(params NewParams) (*Job, error) {
	if !params.ID.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidID, params.ID)
	}

	if !params.Type.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidType, params.Type)
	}

	if err := validatePayload(params.Payload); err != nil {
		return nil, err
	}

	if params.CreatedAt.IsZero() {
		return nil, ErrInvalidCreatedAt
	}

	createdAt := params.CreatedAt.UTC()

	return &Job{
		id:         params.ID,
		jobType:    params.Type,
		status:     StatusPending,
		payload:    clonePayload(params.Payload),
		retryCount: 0,
		maxRetries: DefaultMaxRetries,
		createdAt:  createdAt,
		updatedAt:  createdAt,
	}, nil
}

// Restore reconstructs a Job from trusted persistence data while rechecking
// domain invariants. Repositories should use this instead of bypassing the
// entity's private fields.
func Restore(params RestoreParams) (*Job, error) {
	if !params.ID.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidID, params.ID)
	}

	if !params.Type.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidType, params.Type)
	}

	if !params.Status.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidStatus, params.Status)
	}

	if err := validatePayload(params.Payload); err != nil {
		return nil, err
	}

	if params.RetryCount < 0 || params.MaxRetries <= 0 || params.RetryCount > params.MaxRetries {
		return nil, fmt.Errorf(
			"%w: retry count %d, max retries %d",
			ErrInvalidRetry,
			params.RetryCount,
			params.MaxRetries,
		)
	}

	if params.CreatedAt.IsZero() {
		return nil, ErrInvalidCreatedAt
	}

	if params.UpdatedAt.IsZero() || params.UpdatedAt.Before(params.CreatedAt) {
		return nil, ErrInvalidUpdatedAt
	}

	if params.StartedAt != nil && params.StartedAt.IsZero() {
		return nil, fmt.Errorf("%w: started_at", ErrInvalidEventTime)
	}

	if params.CompletedAt != nil && params.CompletedAt.IsZero() {
		return nil, fmt.Errorf("%w: completed_at", ErrInvalidEventTime)
	}

	return &Job{
		id:          params.ID,
		jobType:     params.Type,
		status:      params.Status,
		payload:     clonePayload(params.Payload),
		retryCount:  params.RetryCount,
		maxRetries:  params.MaxRetries,
		lastError:   cloneStringPointer(params.LastError),
		createdAt:   params.CreatedAt.UTC(),
		updatedAt:   params.UpdatedAt.UTC(),
		startedAt:   cloneTimePointerUTC(params.StartedAt),
		completedAt: cloneTimePointerUTC(params.CompletedAt),
	}, nil
}

func (job *Job) ID() ID {
	return job.id
}

func (job *Job) Type() Type {
	return job.jobType
}

func (job *Job) Status() Status {
	return job.status
}

func (job *Job) Payload() json.RawMessage {
	return clonePayload(job.payload)
}

func (job *Job) RetryCount() int {
	return job.retryCount
}

func (job *Job) MaxRetries() int {
	return job.maxRetries
}

func (job *Job) LastError() (string, bool) {
	if job.lastError == nil {
		return "", false
	}

	return *job.lastError, true
}

func (job *Job) CreatedAt() time.Time {
	return job.createdAt
}

func (job *Job) UpdatedAt() time.Time {
	return job.updatedAt
}

func (job *Job) StartedAt() (time.Time, bool) {
	if job.startedAt == nil {
		return time.Time{}, false
	}

	return *job.startedAt, true
}

func (job *Job) CompletedAt() (time.Time, bool) {
	if job.completedAt == nil {
		return time.Time{}, false
	}

	return *job.completedAt, true
}

func validatePayload(payload json.RawMessage) error {
	trimmedPayload := bytes.TrimSpace(payload)
	if len(trimmedPayload) == 0 {
		return fmt.Errorf("%w: payload is required", ErrInvalidPayload)
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(trimmedPayload, &object); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}

	if object == nil {
		return fmt.Errorf("%w: payload must be a JSON object", ErrInvalidPayload)
	}

	if len(object) == 0 {
		return fmt.Errorf("%w: payload must not be empty", ErrInvalidPayload)
	}

	return nil
}

func clonePayload(payload json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), payload...)
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneTimePointerUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()
	return &cloned
}

func (entity *Job) MarkProcessing(at time.Time) error {
	if entity.status != StatusPending {
		return fmt.Errorf(
			"%w: cannot move job from %s to %s",
			ErrInvalidTransition,
			entity.status,
			StatusProcessing,
		)
	}

	transitionTime, err := entity.transitionTime(at)
	if err != nil {
		return err
	}

	entity.status = StatusProcessing
	entity.startedAt = &transitionTime
	entity.completedAt = nil
	entity.updatedAt = transitionTime

	return nil
}

func (entity *Job) MarkSuccess(at time.Time) error {
	if entity.status != StatusProcessing {
		return fmt.Errorf(
			"%w: cannot move job from %s to %s",
			ErrInvalidTransition,
			entity.status,
			StatusSuccess,
		)
	}

	transitionTime, err := entity.transitionTime(at)
	if err != nil {
		return err
	}

	entity.status = StatusSuccess
	entity.completedAt = &transitionTime
	entity.lastError = nil
	entity.updatedAt = transitionTime

	return nil
}

func (entity *Job) MarkFailed(
	at time.Time,
	reason string,
) error {
	if entity.status != StatusProcessing {
		return fmt.Errorf(
			"%w: cannot move job from %s to %s",
			ErrInvalidTransition,
			entity.status,
			StatusFailed,
		)
	}

	normalizedReason := strings.TrimSpace(reason)
	if normalizedReason == "" {
		return ErrInvalidFailureReason
	}

	transitionTime, err := entity.transitionTime(at)
	if err != nil {
		return err
	}

	entity.status = StatusFailed
	entity.lastError = &normalizedReason
	entity.completedAt = &transitionTime
	entity.updatedAt = transitionTime

	return nil
}

func (entity *Job) transitionTime(
	at time.Time,
) (time.Time, error) {
	if at.IsZero() || at.Before(entity.updatedAt) {
		return time.Time{}, fmt.Errorf(
			"%w: transition time must not be before the last update",
			ErrInvalidEventTime,
		)
	}

	return at.UTC(), nil
}
