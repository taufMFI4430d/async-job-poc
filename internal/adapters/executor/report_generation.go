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

const maximumReportNameLength = 100

type reportGenerationPayload struct {
	Report string `json:"report"`
	Format string `json:"format"`
}

type ReportGenerationHandler struct {
	processingDuration time.Duration
	logger             *slog.Logger
}

var _ Handler = (*ReportGenerationHandler)(nil)

func NewReportGenerationHandler(
	processingDuration time.Duration,
	logger *slog.Logger,
) (*ReportGenerationHandler, error) {
	if processingDuration <= 0 {
		return nil, errors.New(
			"processing duration must be greater than zero",
		)
	}

	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}

	return &ReportGenerationHandler{
		processingDuration: processingDuration,
		logger:             logger,
	}, nil
}

func (handler *ReportGenerationHandler) Handle(
	ctx context.Context,
	entity *job.Job,
) error {
	if entity == nil {
		return errors.New("job must not be nil")
	}

	if entity.Type() != job.TypeReportGeneration {
		return fmt.Errorf(
			"%w: report-generation handler received %s",
			ErrUnexpectedJobType,
			entity.Type(),
		)
	}

	payload, err := decodeReportGenerationPayload(
		entity.Payload(),
	)
	if err != nil {
		return fmt.Errorf(
			"decode report-generation job %s: %w",
			entity.ID(),
			err,
		)
	}

	handler.logger.Info(
		"report generation started",
		slog.String("job_id", entity.ID().String()),
		slog.String("report", payload.Report),
		slog.String("format", payload.Format),
	)

	timer := time.NewTimer(handler.processingDuration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf(
			"generate report for job %s: %w",
			entity.ID(),
			ctx.Err(),
		)

	case <-timer.C:
		handler.logger.Info(
			"report generation completed",
			slog.String(
				"job_id",
				entity.ID().String(),
			),
			slog.String("report", payload.Report),
			slog.String("format", payload.Format),
		)

		return nil
	}
}

func decodeReportGenerationPayload(
	rawPayload json.RawMessage,
) (reportGenerationPayload, error) {
	payload, err :=
		decodeStrictPayload[reportGenerationPayload](
			rawPayload,
		)
	if err != nil {
		return reportGenerationPayload{}, err
	}

	payload.Report = strings.TrimSpace(payload.Report)
	payload.Format = strings.ToLower(
		strings.TrimSpace(payload.Format),
	)

	if err := validateReportGenerationPayload(
		payload,
	); err != nil {
		return reportGenerationPayload{}, err
	}

	return payload, nil
}

func validateReportGenerationPayload(
	payload reportGenerationPayload,
) error {
	if payload.Report == "" {
		return fmt.Errorf(
			"%w: report is required",
			ErrInvalidHandlerPayload,
		)
	}

	if len(payload.Report) > maximumReportNameLength {
		return fmt.Errorf(
			"%w: report must not exceed %d characters",
			ErrInvalidHandlerPayload,
			maximumReportNameLength,
		)
	}

	if payload.Format == "" {
		return fmt.Errorf(
			"%w: format is required",
			ErrInvalidHandlerPayload,
		)
	}

	if !isSupportedReportFormat(payload.Format) {
		return fmt.Errorf(
			"%w: format must be csv, json, or pdf",
			ErrInvalidHandlerPayload,
		)
	}

	return nil
}

func isSupportedReportFormat(format string) bool {
	switch format {
	case "csv", "json", "pdf":
		return true

	default:
		return false
	}
}
