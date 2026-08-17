package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/taufMFI4430d/async-job-poc/internal/platform/requestid"
)

func TestRequestIDMiddlewarePreservesValidClientID(t *testing.T) {
	const clientID = "client-request-123"

	var contextID string
	handler := requestIDMiddleware(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		contextID = requestid.FromContext(request.Context())
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	request.Header.Set(RequestIDHeader, clientID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if contextID != clientID {
		t.Fatalf("expected context request ID %q, got %q", clientID, contextID)
	}
	if response.Header().Get(RequestIDHeader) != clientID {
		t.Fatalf(
			"expected response request ID %q, got %q",
			clientID,
			response.Header().Get(RequestIDHeader),
		)
	}
}

func TestRequestIDMiddlewareGeneratesIDWhenMissingOrInvalid(t *testing.T) {
	for _, incomingID := range []string{"", "invalid request id"} {
		t.Run(incomingID, func(t *testing.T) {
			var contextID string
			handler := requestIDMiddleware(http.HandlerFunc(func(
				writer http.ResponseWriter,
				request *http.Request,
			) {
				contextID = requestid.FromContext(request.Context())
				writer.WriteHeader(http.StatusNoContent)
			}))

			request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
			request.Header.Set(RequestIDHeader, incomingID)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			responseID := response.Header().Get(RequestIDHeader)
			if responseID == "" {
				t.Fatal("expected a generated response request ID")
			}
			if contextID != responseID {
				t.Fatalf("expected context ID %q, got %q", responseID, contextID)
			}
			if incomingID != "" && responseID == incomingID {
				t.Fatal("expected invalid client request ID to be replaced")
			}
		})
	}
}
