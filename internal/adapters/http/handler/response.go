package handler

import (
	"encoding/json"
	"net/http"
)

type errorEnvelope struct {
	Error apiErrorResponse `json:"error"`
}

type apiErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeAPIError(
	writer http.ResponseWriter,
	status int,
	code string,
	message string,
) {
	writeJSON(writer, status, errorEnvelope{
		Error: apiErrorResponse{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(
	writer http.ResponseWriter,
	status int,
	response any,
) {

	writer.WriteHeader(status)

	// A failure here usually means the client disconnected.
	_ = json.NewEncoder(writer).Encode(response)
}
