package clock

import (
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
)

type SystemClock struct{}

var _ ports.Clock = (*SystemClock)(nil)

func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

func (clock *SystemClock) Now() time.Time {
	return time.Now().UTC()
}
