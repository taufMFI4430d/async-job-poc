package job

import (
	"fmt"
	"strings"
)

type Type string

const (
	TypeSendEmail        Type = "send_email"
	TypeReportGeneration Type = "report_generation"
	TypeDataCleanup      Type = "data_cleanup"
)

func ParseType(value string) (Type, error) {
	jobType := Type(strings.ToLower(strings.TrimSpace(value)))
	if !jobType.IsValid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidType, value)
	}

	return jobType, nil
}

func (jobType Type) IsValid() bool {
	switch jobType {
	case TypeSendEmail, TypeReportGeneration, TypeDataCleanup:
		return true
	default:
		return false
	}
}

func (jobType Type) String() string {
	return string(jobType)
}
