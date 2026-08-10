package executor

import "errors"

var (
	ErrUnexpectedJobType = errors.New(
		"unexpected job type",
	)

	ErrInvalidHandlerPayload = errors.New(
		"invalid handler payload",
	)
)
