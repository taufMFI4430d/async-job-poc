package logging

import (
	"errors"
	"io"
	"log/slog"
	"strings"
)

const defaultServiceName = "async-job-poc"

type Options struct {
	Level       string
	Environment string
	ServiceName string
}

func New(output io.Writer, options Options) (*slog.Logger, error) {
	if output == nil {
		return nil, errors.New("log output must not be nil")
	}

	level, err := parseLevel(options.Level)
	if err != nil {
		return nil, err
	}

	serviceName := strings.TrimSpace(options.ServiceName)
	if serviceName == "" {
		serviceName = defaultServiceName
	}

	handler := slog.NewJSONHandler(
		output,
		&slog.HandlerOptions{
			Level:     level,
			AddSource: options.Environment == "development",
		},
	)

	logger := slog.New(handler).With(
		slog.String("service", serviceName),
		slog.String("environment", options.Environment),
	)

	return logger, nil
}

func parseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, errors.New(
			"log level must be one of debug, info, warn, or error",
		)
	}
}
