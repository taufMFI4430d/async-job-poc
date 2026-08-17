package httpadapter

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const indexFile = "index.html"

// UIHandler serves the compiled React application and falls back to index.html
// for browser routes. API and health routes remain owned by the HTTP router.
type UIHandler struct {
	files      fs.FS
	fileServer http.Handler
}

func NewUIHandler(files fs.FS) (*UIHandler, error) {
	if files == nil {
		return nil, errors.New("UI filesystem must not be nil")
	}

	if _, err := fs.Stat(files, indexFile); err != nil {
		return nil, fmt.Errorf("find UI entrypoint: %w", err)
	}

	return &UIHandler{
		files:      files,
		fileServer: http.FileServer(http.FS(files)),
	}, nil
}

func (handler *UIHandler) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	setUISecurityHeaders(writer.Header())

	filePath := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
	if filePath == "." || filePath == "" {
		handler.serveIndex(writer, request)
		return
	}
	if strings.HasPrefix(filePath, "api/") || strings.HasPrefix(filePath, "health/") {
		http.NotFound(writer, request)
		return
	}

	if fileInfo, err := fs.Stat(handler.files, filePath); err == nil && !fileInfo.IsDir() {
		if strings.HasPrefix(filePath, "assets/") {
			writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		handler.fileServer.ServeHTTP(writer, request)
		return
	}

	handler.serveIndex(writer, request)
}

func (handler *UIHandler) serveIndex(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set("Cache-Control", "no-cache")
	indexRequest := request.Clone(request.Context())
	indexRequest.URL.Path = "/"
	handler.fileServer.ServeHTTP(writer, indexRequest)
}

func setUISecurityHeaders(header http.Header) {
	header.Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
}
