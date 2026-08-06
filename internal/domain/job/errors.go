package job

import "errors"

var (
	ErrInvalidID            = errors.New("invalid job ID")
	ErrInvalidType          = errors.New("invalid job type")
	ErrInvalidStatus        = errors.New("invalid job status")
	ErrInvalidPayload       = errors.New("invalid job payload")
	ErrInvalidCreatedAt     = errors.New("invalid job creation time")
	ErrInvalidUpdatedAt     = errors.New("invalid job update time")
	ErrInvalidRetry         = errors.New("invalid job retry values")
	ErrInvalidEventTime     = errors.New("invalid job lifecycle time")
	ErrInvalidTransition    = errors.New("invalid job status transition")
	ErrInvalidFailureReason = errors.New("invalid job failure reason")
)
