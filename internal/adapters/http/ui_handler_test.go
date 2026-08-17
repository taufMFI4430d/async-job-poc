package httpadapter_test

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	httpadapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/http"
)

func TestUIHandlerServesIndexAndStaticAssets(t *testing.T) {
	files := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<main>Job Console</main>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ready')")},
	}
	handler, err := httpadapter.NewUIHandler(files)
	if err != nil {
		t.Fatalf("NewUIHandler() returned an unexpected error: %v", err)
	}

	tests := []struct {
		name          string
		path          string
		expectedBody  string
		expectedCache string
	}{
		{name: "root", path: "/", expectedBody: "<main>Job Console</main>", expectedCache: "no-cache"},
		{name: "asset", path: "/assets/app.js", expectedBody: "console.log('ready')", expectedCache: "public, max-age=31536000, immutable"},
		{name: "browser route fallback", path: "/jobs/example", expectedBody: "<main>Job Console</main>", expectedCache: "no-cache"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", response.Code)
			}
			if response.Body.String() != test.expectedBody {
				t.Fatalf("expected body %q, got %q", test.expectedBody, response.Body.String())
			}
			if response.Header().Get("Cache-Control") != test.expectedCache {
				t.Fatalf("expected Cache-Control %q, got %q", test.expectedCache, response.Header().Get("Cache-Control"))
			}
			if response.Header().Get("Content-Security-Policy") == "" {
				t.Fatal("expected Content-Security-Policy header")
			}
		})
	}
}

func TestNewUIHandlerRequiresEntrypoint(t *testing.T) {
	_, err := httpadapter.NewUIHandler(fstest.MapFS{})
	if err == nil {
		t.Fatal("expected an error when index.html is missing")
	}

	if !errorsIsNotExist(err) {
		t.Fatalf("expected missing-file error, got %v", err)
	}
}

func TestUIHandlerDoesNotMaskUnknownAPIRoute(t *testing.T) {
	handler, err := httpadapter.NewUIHandler(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<main>Job Console</main>")},
	})
	if err != nil {
		t.Fatalf("NewUIHandler() returned an unexpected error: %v", err)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}

func errorsIsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
