package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

type HTTPServer struct {
	server *http.Server
	logger *slog.Logger
}

func NewHTTPServer(
	address string,
	router http.Handler,
	logger *slog.Logger,
) *HTTPServer {
	return &HTTPServer{
		server: &http.Server{
			Addr:              address,
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			ErrorLog: slog.NewLogLogger(
				logger.Handler(),
				slog.LevelError,
			),
		},
		logger: logger,
	}
}

func (httpServer *HTTPServer) Run(ctx context.Context) error {
	serverErrors := make(chan error, 1)

	go func() {
		httpServer.logger.Info(
			"http server listening",
			slog.String("address", httpServer.server.Addr),
		)

		err := httpServer.server.ListenAndServe()

		if errors.Is(err, http.ErrServerClosed) {
			serverErrors <- nil
			return
		}

		serverErrors <- err
	}()

	select {
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("run HTTP server: %w", err)
		}

		return nil

	case <-ctx.Done():
		httpServer.logger.Info("HTTP server shutdown requested")
	}

	shutdownContext, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()

	if err := httpServer.server.Shutdown(shutdownContext); err != nil {
		closeErr := httpServer.server.Close()

		return errors.Join(
			fmt.Errorf("gracefully shut down HTTP server: %w", err),
			closeErr,
		)
	}

	if err := <-serverErrors; err != nil {
		return fmt.Errorf("stop HTTP server: %w", err)
	}

	httpServer.logger.Info("HTTP server stopped gracefully")

	return nil
}
