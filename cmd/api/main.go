package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	httpadapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/http"
	httphandler "github.com/taufMFI4430d/async-job-poc/internal/adapters/http/handler"
	mysqladapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/mysql"
	redisadapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/redis"
	"github.com/taufMFI4430d/async-job-poc/internal/application/usecase"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/cache"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/clock"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/config"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/database"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/idgen"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/logging"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/server"
)

func main() {
	os.Exit(run())
}

func run() int {
	bootstrapLogger := slog.New(
		slog.NewJSONHandler(os.Stderr, nil),
	)

	cfg, err := config.Load()
	if err != nil {
		bootstrapLogger.Error(
			"failed to load application configuration",
			slog.Any("error", err),
		)
		return 1
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
		return 1
	}

	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	mysqlDB, err := database.OpenMySQL(ctx, database.MySQLOptions{
		Host:     cfg.MySQL.Host,
		Port:     cfg.MySQL.Port,
		Database: cfg.MySQL.Database,
		User:     cfg.MySQL.User,
		Password: cfg.MySQL.Password,
	})
	if err != nil {
		logger.Error(
			"failed to initialize MySQL",
			slog.Any("error", err),
		)
		return 1
	}

	defer func() {
		if err := mysqlDB.Close(); err != nil {
			logger.Error(
				"failed to close MySQL connection",
				slog.Any("error", err),
			)
		}
	}()

	logger.Info("MySQL connection established")

	redisConnection, err := cache.OpenRedis(
		ctx,
		cache.RedisOptions{
			Host:     cfg.Redis.Host,
			Port:     cfg.Redis.Port,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		},
	)
	if err != nil {
		logger.Error(
			"failed to initialize Redis",
			slog.Any("error", err),
		)
		return 1
	}

	defer func() {
		if err := redisConnection.Close(); err != nil {
			logger.Error(
				"failed to close Redis connection",
				slog.Any("error", err),
			)
		}
	}()

	logger.Info(
		"Redis connection established",
		slog.Int("database", cfg.Redis.DB),
	)

	jobRepository, err := mysqladapter.NewJobRepository(mysqlDB.GORM())
	if err != nil {
		logger.Error(
			"failed to initialize job repository",
			slog.Any("error", err),
		)
		return 1
	}

	jobQueue, err := redisadapter.NewJobQueue(
		redisConnection.Client(),
		cfg.Redis.QueueName,
	)
	if err != nil {
		logger.Error(
			"failed to initialize Redis job queue",
			slog.Any("error", err),
		)
		return 1
	}

	lifecycleObserver, err := logging.NewJobLifecycleObserver(logger)
	if err != nil {
		logger.Error(
			"failed to initialize job lifecycle observer",
			slog.Any("error", err),
		)
		return 1
	}

	createJob, err := usecase.NewCreateJob(
		jobRepository,
		jobQueue,
		idgen.NewUUIDGenerator(),
		clock.NewSystemClock(),
		lifecycleObserver,
	)
	if err != nil {
		logger.Error(
			"failed to initialize create-job use case",
			slog.Any("error", err),
		)
		return 1
	}

	getJob, err := usecase.NewGetJob(jobRepository)
	if err != nil {
		logger.Error(
			"failed to initialize get-job use case",
			slog.Any("error", err),
		)
		return 1
	}

	jobHandler, err := httphandler.NewJobHandler(createJob, getJob, logger)
	if err != nil {
		logger.Error(
			"failed to initialize job handler",
			slog.Any("error", err),
		)
		return 1
	}

	uiHandler, err := httpadapter.NewUIHandler(
		os.DirFS(cfg.UI.AssetsDirectory),
	)
	if err != nil {
		logger.Error(
			"failed to initialize UI handler",
			slog.String("assets_directory", cfg.UI.AssetsDirectory),
			slog.Any("error", err),
		)
		return 1
	}

	router := httpadapter.NewRouter(
		jobHandler,
		uiHandler,
		mysqlDB,
		redisConnection,
	)

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
		return 1
	}

	logger.Info("application stopped")
	return 0
}
