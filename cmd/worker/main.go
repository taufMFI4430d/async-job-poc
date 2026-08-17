package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	executoradapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/executor"
	mysqladapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/mysql"
	redisadapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/redis"
	workeradapter "github.com/taufMFI4430d/async-job-poc/internal/adapters/worker"
	"github.com/taufMFI4430d/async-job-poc/internal/application/usecase"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/cache"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/clock"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/config"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/database"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/logging"
)

const handlerProcessingDuration = 2 * time.Second

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
			"failed to load worker configuration",
			slog.Any("error", err),
		)
		return 1
	}

	logger, err := logging.New(
		os.Stdout,
		logging.Options{
			Level:       cfg.Log.Level,
			Environment: cfg.App.Environment,
			ServiceName: "async-job-worker",
		},
	)
	if err != nil {
		bootstrapLogger.Error(
			"failed to initialize worker logger",
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

	mysqlDB, err := database.OpenMySQL(
		ctx,
		database.MySQLOptions{
			Host:     cfg.MySQL.Host,
			Port:     cfg.MySQL.Port,
			Database: cfg.MySQL.Database,
			User:     cfg.MySQL.User,
			Password: cfg.MySQL.Password,
		},
	)
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

	jobRepository, err := mysqladapter.NewJobRepository(
		mysqlDB.GORM(),
	)
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

	sendEmailHandler, err :=
		executoradapter.NewSendEmailHandler(
			handlerProcessingDuration,
			logger,
		)
	if err != nil {
		logger.Error(
			"failed to initialize send-email handler",
			slog.Any("error", err),
		)
		return 1
	}

	reportGenerationHandler, err :=
		executoradapter.NewReportGenerationHandler(
			handlerProcessingDuration,
			logger,
		)
	if err != nil {
		logger.Error(
			"failed to initialize report-generation handler",
			slog.Any("error", err),
		)
		return 1
	}

	dataCleanupHandler, err :=
		executoradapter.NewDataCleanupHandler(
			handlerProcessingDuration,
			logger,
		)
	if err != nil {
		logger.Error(
			"failed to initialize data-cleanup handler",
			slog.Any("error", err),
		)
		return 1
	}

	jobExecutor, err := executoradapter.NewDispatcher(
		map[job.Type]executoradapter.Handler{
			job.TypeSendEmail: sendEmailHandler,

			job.TypeReportGeneration: reportGenerationHandler,

			job.TypeDataCleanup: dataCleanupHandler,
		},
	)
	if err != nil {
		logger.Error(
			"failed to initialize job executor dispatcher",
			slog.Any("error", err),
		)
		return 1
	}

	processJob, err := usecase.NewProcessJob(
		jobRepository,
		jobQueue,
		jobExecutor,
		clock.NewSystemClock(),
		lifecycleObserver,
	)
	if err != nil {
		logger.Error(
			"failed to initialize process-job use case",
			slog.Any("error", err),
		)
		return 1
	}

	workerPool, err := workeradapter.NewPool(
		jobQueue,
		processJob,
		logger,
	)
	if err != nil {
		logger.Error(
			"failed to initialize worker pool",
			slog.Any("error", err),
		)
		return 1
	}

	logger.Info(
		"worker application starting",
		slog.String("queue", cfg.Redis.QueueName),
		slog.Int("worker_count", workeradapter.WorkerCount),
	)

	if err := workerPool.Run(ctx); err != nil {
		logger.Error(
			"worker pool stopped unexpectedly",
			slog.Any("error", err),
		)
		return 1
	}

	logger.Info("worker application stopped")
	return 0
}
