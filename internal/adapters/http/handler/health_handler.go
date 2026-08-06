package handler

import (
	"context"
	"net/http"
	"time"
)

const readinessTimeout = 2 * time.Second

type ReadinessChecker interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	readinessCheckers []ReadinessChecker
}

func NewHealthHandler(
	readinessCheckers ...ReadinessChecker,
) *HealthHandler {
	return &HealthHandler{
		readinessCheckers: readinessCheckers,
	}
}

func (handler *HealthHandler) Live(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writeJSON(writer, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (handler *HealthHandler) Ready(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if len(handler.readinessCheckers) == 0 {
		writeNotReady(writer)
		return
	}

	ctx, cancel := context.WithTimeout(
		request.Context(),
		readinessTimeout,
	)
	defer cancel()

	for _, readinessChecker := range handler.readinessCheckers {
		if readinessChecker == nil {
			writeNotReady(writer)
			return
		}

		if err := readinessChecker.Ping(ctx); err != nil {
			writeNotReady(writer)
			return
		}
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func writeNotReady(writer http.ResponseWriter) {
	writeJSON(
		writer,
		http.StatusServiceUnavailable,
		map[string]string{
			"status": "not_ready",
		},
	)
}
