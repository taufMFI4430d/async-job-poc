package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	httpadapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/http"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/config"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/logging"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/server"
)

func main() {
	bootstrapLogger := slog.New(
		slog.NewJSONHandler(os.Stderr, nil),
	)

	cfg, err := config.Load()
	if err != nil {
		bootstrapLogger.Error(
			"failed to load application configuration",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	logger, err := logging.New(
		os.Stdout,
		logging.Options{
			Level:       cfg.Log.Level,
			Environment: cfg.App.Environment,
			ServiceName: "async-job-api",
		},
	)
	if err != nil {
		bootstrapLogger.Error(
			"failed to initialize application logger",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	router := httpadapter.NewRouter()

	httpServer := server.NewHTTPServer(
		cfg.HTTP.Address,
		router,
		logger,
	)

	logger.Info("application starting")

	if err := httpServer.Run(ctx); err != nil {
		logger.Error(
			"application stopped unexpectedly",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	logger.Info("application stopped")
}
