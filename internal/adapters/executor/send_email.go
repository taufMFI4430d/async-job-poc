package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
	platformlogging "github.com/taufMFI4430d/async-job-poc/internal/platform/logging"
)

const maximumEmailSubjectLength = 200

type sendEmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type SendEmailHandler struct {
	processingDuration time.Duration
	logger             *slog.Logger
}

var _ Handler = (*SendEmailHandler)(nil)

func NewSendEmailHandler(
	processingDuration time.Duration,
	logger *slog.Logger,
) (*SendEmailHandler, error) {
	if processingDuration <= 0 {
		return nil, errors.New(
			"processing duration must be greater than zero",
		)
	}

	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}

	return &SendEmailHandler{
		processingDuration: processingDuration,
		logger:             logger,
	}, nil
}

func (handler *SendEmailHandler) Handle(
	ctx context.Context,
	entity *job.Job,
) error {
	if entity == nil {
		return errors.New("job must not be nil")
	}

	if entity.Type() != job.TypeSendEmail {
		return fmt.Errorf(
			"%w: send-email handler received %s",
			ErrUnexpectedJobType,
			entity.Type(),
		)
	}

	payload, err := decodeSendEmailPayload(
		entity.Payload(),
	)
	if err != nil {
		return fmt.Errorf(
			"decode send-email job %s: %w",
			entity.ID(),
			err,
		)
	}

	attributes := platformlogging.JobAttributes(entity)
	attributes = append(
		attributes,
		slog.String("recipient", payload.To),
	)

	handler.logger.LogAttrs(
		ctx,
		slog.LevelInfo,
		"email delivery started",
		attributes...,
	)

	timer := time.NewTimer(handler.processingDuration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf(
			"deliver email for job %s: %w",
			entity.ID(),
			ctx.Err(),
		)

	case <-timer.C:
		handler.logger.LogAttrs(
			ctx,
			slog.LevelInfo,
			"email delivery completed",
			attributes...,
		)

		return nil
	}
}

func decodeSendEmailPayload(
	rawPayload json.RawMessage,
) (sendEmailPayload, error) {
	payload, err := decodeStrictPayload[sendEmailPayload](
		rawPayload,
	)
	if err != nil {
		return sendEmailPayload{}, err
	}

	payload.To = strings.TrimSpace(payload.To)
	payload.Subject = strings.TrimSpace(payload.Subject)
	payload.Body = strings.TrimSpace(payload.Body)

	if err := validateSendEmailPayload(payload); err != nil {
		return sendEmailPayload{}, err
	}

	return payload, nil
}

func validateSendEmailPayload(
	payload sendEmailPayload,
) error {
	if payload.To == "" {
		return fmt.Errorf(
			"%w: to is required",
			ErrInvalidHandlerPayload,
		)
	}

	parsedAddress, err := mail.ParseAddress(payload.To)
	if err != nil || parsedAddress.Address != payload.To {
		return fmt.Errorf(
			"%w: to must be a valid email address",
			ErrInvalidHandlerPayload,
		)
	}

	if payload.Subject == "" {
		return fmt.Errorf(
			"%w: subject is required",
			ErrInvalidHandlerPayload,
		)
	}

	if len(payload.Subject) > maximumEmailSubjectLength {
		return fmt.Errorf(
			"%w: subject must not exceed %d characters",
			ErrInvalidHandlerPayload,
			maximumEmailSubjectLength,
		)
	}

	if payload.Body == "" {
		return fmt.Errorf(
			"%w: body is required",
			ErrInvalidHandlerPayload,
		)
	}

	return nil
}
