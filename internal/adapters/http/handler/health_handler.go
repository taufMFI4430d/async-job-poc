package handler

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
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
	_ *http.Request,
) {
	// MySQL readiness will be added when the database adapter is created.
	writeJSON(writer, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func writeJSON(
	writer http.ResponseWriter,
	status int,
	response any,
) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)

	// A failure here usually means the client disconnected.
	_ = json.NewEncoder(writer).Encode(response)
}
