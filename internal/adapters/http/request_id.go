package httpadapter

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"

	"github.com/taufMFI4430d/async-job-poc/internal/platform/requestid"
)

const RequestIDHeader = "X-Request-ID"

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		id := strings.TrimSpace(request.Header.Get(RequestIDHeader))
		if !isValidRequestID(id) {
			generatedID, err := newRequestID()
			if err != nil {
				http.Error(writer, "failed to create request ID", http.StatusInternalServerError)
				return
			}
			id = generatedID
		}

		writer.Header().Set(RequestIDHeader, id)
		ctx := requestid.WithContext(request.Context(), id)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func isValidRequestID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}

	for _, character := range id {
		if !isAllowedRequestIDCharacter(character) {
			return false
		}
	}

	return true
}

func isAllowedRequestIDCharacter(character rune) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' ||
		strings.ContainsRune("-_.", character)
}

func newRequestID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate request ID: %w", err)
	}

	bytes[6] = bytes[6]&0x0f | 0x40
	bytes[8] = bytes[8]&0x3f | 0x80

	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16],
	), nil
}
