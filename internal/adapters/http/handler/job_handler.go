package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/application/usecase"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
	platformlogging "github.com/taufMFI4430d/async-job-poc/internal/platform/logging"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/requestid"
)

const maxCreateJobBodyBytes = 1 << 20

type CreateJobExecutor interface {
	Execute(context.Context, usecase.CreateJobInput) (*job.Job, error)
}

type GetJobExecutor interface {
	Execute(context.Context, usecase.GetJobInput) (*job.Job, error)
}

type JobHandler struct {
	createJob CreateJobExecutor
	getJob    GetJobExecutor
	logger    *slog.Logger
}

type createJobRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type jobResponse struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Status      string          `json:"status"`
	Payload     json.RawMessage `json:"payload"`
	RetryCount  int             `json:"retry_count"`
	MaxRetries  int             `json:"max_retries"`
	LastError   *string         `json:"last_error"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	StartedAt   *time.Time      `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at"`
}

func NewJobHandler(
	createJob CreateJobExecutor,
	getJob GetJobExecutor,
	logger *slog.Logger,
) (*JobHandler, error) {
	if createJob == nil {
		return nil, errors.New("create-job use case must not be nil")
	}

	if getJob == nil {
		return nil, errors.New("get-job use case must not be nil")
	}

	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}

	return &JobHandler{
		createJob: createJob,
		getJob:    getJob,
		logger:    logger,
	}, nil
}

func (handler *JobHandler) Create(
	writer http.ResponseWriter,
	request *http.Request,
) {
	requestBody, requestError := decodeCreateJobRequest(writer, request)
	if requestError != nil {
		handler.logger.Warn(
			"job request rejected",
			platformlogging.RequestIDAttribute(requestid.FromContext(request.Context())),
			slog.String("operation", "create_job"),
			slog.String("error_code", requestError.code),
		)
		writeAPIError(
			writer,
			requestError.status,
			requestError.code,
			requestError.message,
		)
		return
	}

	entity, err := handler.createJob.Execute(
		request.Context(),
		usecase.CreateJobInput{
			Type:    requestBody.Type,
			Payload: requestBody.Payload,
		},
	)
	if err != nil {
		handler.writeApplicationError(request.Context(), writer, "create_job", err)
		return
	}

	attributes := platformlogging.JobAttributes(entity)
	attributes = append(attributes,
		platformlogging.RequestIDAttribute(requestid.FromContext(request.Context())),
	)
	handler.logger.LogAttrs(
		request.Context(),
		slog.LevelInfo,
		"job accepted",
		attributes...,
	)

	writer.Header().Set(
		"Location",
		"/api/v1/jobs/"+entity.ID().String(),
	)
	writeJSON(writer, http.StatusAccepted, jobResponseFromDomain(entity))
}

func (handler *JobHandler) GetByID(
	writer http.ResponseWriter,
	request *http.Request,
) {
	entity, err := handler.getJob.Execute(
		request.Context(),
		usecase.GetJobInput{ID: request.PathValue("jobID")},
	)
	if err != nil {
		handler.writeApplicationError(request.Context(), writer, "get_job", err)
		return
	}

	attributes := platformlogging.JobAttributes(entity)
	attributes = append(attributes,
		platformlogging.RequestIDAttribute(requestid.FromContext(request.Context())),
	)
	handler.logger.LogAttrs(
		request.Context(),
		slog.LevelInfo,
		"job status returned",
		attributes...,
	)

	writeJSON(writer, http.StatusOK, jobResponseFromDomain(entity))
}

type requestDecodeError struct {
	status  int
	code    string
	message string
}

func decodeCreateJobRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (createJobRequest, *requestDecodeError) {
	contentType := request.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return createJobRequest{}, &requestDecodeError{
			status:  http.StatusUnsupportedMediaType,
			code:    "unsupported_media_type",
			message: "Content-Type must be application/json",
		}
	}

	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		maxCreateJobBodyBytes,
	)

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	var decoded createJobRequest
	if err := decoder.Decode(&decoded); err != nil {
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.Is(err, io.EOF):
			return createJobRequest{}, &requestDecodeError{
				status:  http.StatusBadRequest,
				code:    "invalid_request",
				message: "request body is required",
			}
		case errors.As(err, &maxBytesError):
			return createJobRequest{}, &requestDecodeError{
				status:  http.StatusRequestEntityTooLarge,
				code:    "request_too_large",
				message: "request body exceeds 1 MiB",
			}
		default:
			return createJobRequest{}, &requestDecodeError{
				status:  http.StatusBadRequest,
				code:    "invalid_request",
				message: "request body must contain valid job JSON",
			}
		}
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return createJobRequest{}, &requestDecodeError{
			status:  http.StatusBadRequest,
			code:    "invalid_request",
			message: "request body must contain one JSON object",
		}
	}

	return decoded, nil
}

func (handler *JobHandler) writeApplicationError(
	ctx context.Context,
	writer http.ResponseWriter,
	operation string,
	err error,
) {
	switch {
	case errors.Is(err, job.ErrInvalidType):
		writeAPIError(
			writer,
			http.StatusBadRequest,
			"validation_error",
			"job type is not supported",
		)
	case errors.Is(err, job.ErrInvalidPayload):
		writeAPIError(
			writer,
			http.StatusBadRequest,
			"validation_error",
			"payload must be a non-empty JSON object",
		)
	case errors.Is(err, job.ErrInvalidID):
		writeAPIError(
			writer,
			http.StatusBadRequest,
			"validation_error",
			"job ID must be a valid UUID",
		)
	case errors.Is(err, ports.ErrJobNotFound):
		writeAPIError(
			writer,
			http.StatusNotFound,
			"job_not_found",
			"job was not found",
		)
	case errors.Is(err, ports.ErrJobAlreadyExists):
		writeAPIError(
			writer,
			http.StatusConflict,
			"job_already_exists",
			"job already exists",
		)
	case errors.Is(err, ports.ErrJobQueueUnavailable):
		handler.logger.LogAttrs(
			ctx,
			slog.LevelError,
			"job could not be published to Redis",
			slog.String("operation", operation),
			platformlogging.RequestIDAttribute(requestid.FromContext(ctx)),
			platformlogging.ErrorAttribute(err),
		)

		writeAPIError(
			writer,
			http.StatusServiceUnavailable,
			"job_queue_unavailable",
			"job queue is temporarily unavailable",
		)
	default:
		handler.logger.LogAttrs(
			ctx,
			slog.LevelError,
			"job request failed",
			slog.String("operation", operation),
			platformlogging.RequestIDAttribute(requestid.FromContext(ctx)),
			platformlogging.ErrorAttribute(err),
		)
		writeAPIError(
			writer,
			http.StatusInternalServerError,
			"internal_error",
			"an unexpected error occurred",
		)
	}
}

func jobResponseFromDomain(entity *job.Job) jobResponse {
	response := jobResponse{
		ID:         entity.ID().String(),
		Type:       entity.Type().String(),
		Status:     entity.Status().String(),
		Payload:    entity.Payload(),
		RetryCount: entity.RetryCount(),
		MaxRetries: entity.MaxRetries(),
		CreatedAt:  entity.CreatedAt(),
		UpdatedAt:  entity.UpdatedAt(),
	}

	if lastError, exists := entity.LastError(); exists {
		response.LastError = &lastError
	}

	if startedAt, exists := entity.StartedAt(); exists {
		response.StartedAt = &startedAt
	}

	if completedAt, exists := entity.CompletedAt(); exists {
		response.CompletedAt = &completedAt
	}

	return response
}
