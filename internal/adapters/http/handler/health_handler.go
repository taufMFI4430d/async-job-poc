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
	readinessChecker ReadinessChecker
}

func NewHealthHandler(readinessChecker ReadinessChecker) *HealthHandler {
	return &HealthHandler{readinessChecker: readinessChecker}
}

func (h *HealthHandler) Live(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writeJSON(writer, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *HealthHandler) Ready(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if h.readinessChecker == nil {
		writeJSON(writer, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
		})
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), readinessTimeout)
	defer cancel()

	if err := h.readinessChecker.Ping(ctx); err != nil {
		writeJSON(writer, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
		})
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"status": "ready",
	})
}
