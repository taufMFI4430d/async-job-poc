package job

import (
	"fmt"
	"strings"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusSuccess    Status = "success"
	StatusFailed     Status = "failed"
)

func ParseStatus(value string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(value)))
	if !status.IsValid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidStatus, value)
	}

	return status, nil
}

func (status Status) IsValid() bool {
	switch status {
	case StatusPending, StatusProcessing, StatusSuccess, StatusFailed:
		return true
	default:
		return false
	}
}

func (status Status) String() string {
	return string(status)
}
