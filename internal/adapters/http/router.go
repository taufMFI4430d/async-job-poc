package httpadapter

import (
	"net/http"

	"github.com/taufMFI4430d/async-job-poc/internal/adapters/http/handler"
)

func NewRouter(
	jobHandler *handler.JobHandler,
	uiHandler http.Handler,
	readinessCheckers ...handler.ReadinessChecker,
) http.Handler {
	router := http.NewServeMux()

	healthHandler := handler.NewHealthHandler(
		readinessCheckers...,
	)

	router.HandleFunc("GET /health/live", healthHandler.Live)
	router.HandleFunc("GET /health/ready", healthHandler.Ready)
	router.HandleFunc("POST /api/v1/jobs", jobHandler.Create)
	router.HandleFunc("GET /api/v1/jobs/{jobID}", jobHandler.GetByID)
	router.Handle("GET /", uiHandler)

	return requestIDMiddleware(router)
}
