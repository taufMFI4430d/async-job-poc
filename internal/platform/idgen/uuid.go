package idgen

import (
	"crypto/rand"
	"fmt"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

type UUIDGenerator struct{}

var _ ports.IDGenerator = (*UUIDGenerator)(nil)

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (generator *UUIDGenerator) NewID() (job.ID, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate random UUID bytes: %w", err)
	}

	// Set the RFC 4122 version (4) and variant bits.
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80

	rawID := fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		value[0:4],
		value[4:6],
		value[6:8],
		value[8:10],
		value[10:16],
	)

	jobID, err := job.ParseID(rawID)
	if err != nil {
		return "", fmt.Errorf("parse generated UUID: %w", err)
	}

	return jobID, nil
}
