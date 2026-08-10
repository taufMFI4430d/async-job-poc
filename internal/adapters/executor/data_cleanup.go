package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
)

const (
	maximumCleanupScopeLength = 100
	maximumCleanupAgeDays     = 3650
)

type dataCleanupPayload struct {
	Scope         string `json:"scope"`
	OlderThanDays int    `json:"older_than_days"`
}

type DataCleanupHandler struct {
	processingDuration time.Duration
	logger             *slog.Logger
}

var _ Handler = (*DataCleanupHandler)(nil)

func NewDataCleanupHandler(
	processingDuration time.Duration,
	logger *slog.Logger,
) (*DataCleanupHandler, error) {
	if processingDuration <= 0 {
		return nil, errors.New(
			"processing duration must be greater than zero",
		)
	}

	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}

	return &DataCleanupHandler{
		processingDuration: processingDuration,
		logger:             logger,
	}, nil
}

func (handler *DataCleanupHandler) Handle(
	ctx context.Context,
	entity *job.Job,
) error {
	if entity == nil {
		return errors.New("job must not be nil")
	}

	if entity.Type() != job.TypeDataCleanup {
		return fmt.Errorf(
			"%w: data-cleanup handler received %s",
			ErrUnexpectedJobType,
			entity.Type(),
		)
	}

	payload, err := decodeDataCleanupPayload(
		entity.Payload(),
	)
	if err != nil {
		return fmt.Errorf(
			"decode data-cleanup job %s: %w",
			entity.ID(),
			err,
		)
	}

	handler.logger.Info(
		"data cleanup started",
		slog.String("job_id", entity.ID().String()),
		slog.String("scope", payload.Scope),
		slog.Int(
			"older_than_days",
			payload.OlderThanDays,
		),
	)

	timer := time.NewTimer(handler.processingDuration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf(
			"clean data for job %s: %w",
			entity.ID(),
			ctx.Err(),
		)

	case <-timer.C:
		handler.logger.Info(
			"data cleanup completed",
			slog.String(
				"job_id",
				entity.ID().String(),
			),
			slog.String("scope", payload.Scope),
			slog.Int(
				"older_than_days",
				payload.OlderThanDays,
			),
		)

		return nil
	}
}

func decodeDataCleanupPayload(
	rawPayload json.RawMessage,
) (dataCleanupPayload, error) {
	payload, err := decodeStrictPayload[dataCleanupPayload](
		rawPayload,
	)
	if err != nil {
		return dataCleanupPayload{}, err
	}

	payload.Scope = strings.TrimSpace(payload.Scope)

	if err := validateDataCleanupPayload(payload); err != nil {
		return dataCleanupPayload{}, err
	}

	return payload, nil
}

func validateDataCleanupPayload(
	payload dataCleanupPayload,
) error {
	if payload.Scope == "" {
		return fmt.Errorf(
			"%w: scope is required",
			ErrInvalidHandlerPayload,
		)
	}

	if len(payload.Scope) > maximumCleanupScopeLength {
		return fmt.Errorf(
			"%w: scope must not exceed %d characters",
			ErrInvalidHandlerPayload,
			maximumCleanupScopeLength,
		)
	}

	if payload.OlderThanDays <= 0 {
		return fmt.Errorf(
			"%w: older_than_days must be greater than zero",
			ErrInvalidHandlerPayload,
		)
	}

	if payload.OlderThanDays > maximumCleanupAgeDays {
		return fmt.Errorf(
			"%w: older_than_days must not exceed %d",
			ErrInvalidHandlerPayload,
			maximumCleanupAgeDays,
		)
	}

	return nil
}
