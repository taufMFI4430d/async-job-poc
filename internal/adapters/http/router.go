package httpadapter

import (
	"net/http"

	"github.com/taufMFI4430d/async-job-poc/internal/adapters/http/handler"
)

func NewRouter() http.Handler {
	router := http.NewServeMux()

	healthHandler := handler.NewHealthHandler()

	router.HandleFunc("GET /health/live", healthHandler.Live)
	router.HandleFunc("GET /health/ready", healthHandler.Ready)

	return router
}
