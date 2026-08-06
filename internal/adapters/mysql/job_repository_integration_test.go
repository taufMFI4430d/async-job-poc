//go:build integration

package mysql

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/taufMFI4430d/async-job-poc/internal/application/ports"
	"github.com/taufMFI4430d/async-job-poc/internal/domain/job"
	"github.com/taufMFI4430d/async-job-poc/internal/platform/database"
)

func TestJobRepositoryAgainstMySQL(t *testing.T) {
	requiredEnvironment := []string{
		"MYSQL_HOST",
		"MYSQL_PORT",
		"MYSQL_DATABASE",
		"MYSQL_USER",
		"MYSQL_PASSWORD",
	}

	for _, name := range requiredEnvironment {
		if os.Getenv(name) == "" {
			t.Skipf("%s is required for the MySQL integration test", name)
		}
	}

	mysqlPort, err := strconv.Atoi(os.Getenv("MYSQL_PORT"))
	if err != nil {
		t.Fatalf("parse MYSQL_PORT: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	connection, err := database.OpenMySQL(ctx, database.MySQLOptions{
		Host:     os.Getenv("MYSQL_HOST"),
		Port:     mysqlPort,
		Database: os.Getenv("MYSQL_DATABASE"),
		User:     os.Getenv("MYSQL_USER"),
		Password: os.Getenv("MYSQL_PASSWORD"),
	})
	if err != nil {
		t.Fatalf("open MySQL: %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close MySQL: %v", err)
		}
	})

	repository, err := NewJobRepository(connection.GORM())
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	jobID := randomJobID(t)
	t.Cleanup(func() {
		result := connection.GORM().
			Where("id = ?", jobID.String()).
			Delete(&jobRecord{})
		if result.Error != nil {
			t.Errorf("delete integration-test job: %v", result.Error)
		}
	})

	entity, err := job.New(job.NewParams{
		ID:        jobID,
		Type:      job.TypeDataCleanup,
		Payload:   json.RawMessage(`{"scope":"integration-test"}`),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create domain job: %v", err)
	}

	if err := repository.Create(ctx, entity); err != nil {
		t.Fatalf("persist job: %v", err)
	}

	if err := repository.Create(ctx, entity); !errors.Is(err, ports.ErrJobAlreadyExists) {
		t.Fatalf("expected ErrJobAlreadyExists, got %v", err)
	}

	persisted, err := repository.GetByID(ctx, jobID)
	if err != nil {
		t.Fatalf("get persisted job: %v", err)
	}

	if persisted.ID() != entity.ID() || persisted.Type() != entity.Type() {
		t.Fatalf("persisted job does not match original job")
	}

	missingID := randomJobID(t)
	_, err = repository.GetByID(ctx, missingID)
	if !errors.Is(err, ports.ErrJobNotFound) {
		t.Fatalf("expected ErrJobNotFound, got %v", err)
	}

	processingAt := time.Now().UTC().Add(time.Second)

	if err := persisted.MarkProcessing(processingAt); err != nil {
		t.Fatalf("mark persisted job as processing: %v", err)
	}

	if err := repository.Update(ctx, persisted); err != nil {
		t.Fatalf("update persisted job: %v", err)
	}

	updated, err := repository.GetByID(ctx, jobID)
	if err != nil {
		t.Fatalf("get updated job: %v", err)
	}

	if updated.Status() != job.StatusProcessing {
		t.Fatalf(
			"expected processing status, got %s",
			updated.Status(),
		)
	}

	startedAt, exists := updated.StartedAt()
	if !exists {
		t.Fatal("expected updated job to have started_at")
	}

	if !startedAt.Equal(processingAt) {
		t.Fatalf(
			"expected started_at %v, got %v",
			processingAt,
			startedAt,
		)
	}
}

func randomJobID(t *testing.T) job.ID {
	t.Helper()

	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		t.Fatalf("generate random job ID: %v", err)
	}

	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80

	rawID := fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		value[0:4],
		value[4:6],
		value[6:8],
		value[8:10],
		value[10:16],
	)

	jobID, err := job.ParseID(rawID)
	if err != nil {
		t.Fatalf("parse generated job ID: %v", err)
	}

	return jobID
}
