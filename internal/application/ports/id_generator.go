package ports

import "github.com/taufMFI4430d/async-job-poc/internal/domain/job"

type IDGenerator interface {
	NewID() (job.ID, error)
}
