package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func TestImportJobStoreClaimNext(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create database mock: %v", err)
	}
	defer db.Close()

	store := NewImportJobStore(db)

	createdAt := time.Date(
		2026,
		time.August,
		17,
		10,
		0,
		0,
		0,
		time.UTC,
	)
	updatedAt := createdAt.Add(time.Minute)
	startedAt := updatedAt

	rows := pgxmock.NewRows([]string{
		"id",
		"recap_id",
		"file_path",
		"status",
		"attempts",
		"last_error",
		"created_at",
		"updated_at",
		"started_at",
		"completed_at",
	}).AddRow(
		int64(1),
		int64(2),
		"/uploads/statement.pdf",
		recap.ImportJobStatusProcessing,
		1,
		nil,
		createdAt,
		updatedAt,
		&startedAt,
		nil,
	)

	db.ExpectQuery(`UPDATE recap_import_jobs .* RETURNING`).
		WithArgs(
			string(recap.ImportJobStatusProcessing),
			string(recap.ImportJobStatusQueued),
		).
		WillReturnRows(rows)

	job, err := store.ClaimNext(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if job == nil {
		t.Fatal("expected a claimed job, got nil")
	}

	if job.ID != 1 {
		t.Errorf("expected job ID %d, got %d", 1, job.ID)
	}

	if job.RecapID != 2 {
		t.Errorf(
			"expected recap ID %d, got %d",
			2,
			job.RecapID,
		)
	}

	if job.Status != recap.ImportJobStatusProcessing {
		t.Errorf(
			"expected status %q, got %q",
			recap.ImportJobStatusProcessing,
			job.Status,
		)
	}

	if job.Attempts != 1 {
		t.Errorf(
			"expected attempts %d, got %d",
			1,
			job.Attempts,
		)
	}

	if job.StartedAt == nil {
		t.Error("expected started_at, got nil")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func TestImportJobStoreClaimNextEmpty(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create database mock: %v", err)
	}
	defer db.Close()

	store := NewImportJobStore(db)

	db.ExpectQuery(`UPDATE recap_import_jobs .* RETURNING`).
		WithArgs(
			string(recap.ImportJobStatusProcessing),
			string(recap.ImportJobStatusQueued),
		).
		WillReturnError(pgx.ErrNoRows)

	job, err := store.ClaimNext(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if job != nil {
		t.Errorf("expected no job, got %+v", job)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func TestImportJobStoreMarkCompleted(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create database mock: %v", err)
	}
	defer db.Close()

	store := NewImportJobStore(db)

	db.ExpectExec(`WITH completed_job AS .* UPDATE recaps`).
		WithArgs(
			int64(1),
			string(recap.ImportJobStatusCompleted),
			string(recap.ImportJobStatusProcessing),
		).
		WillReturnResult(
			pgxmock.NewResult("UPDATE", 1),
		)

	err = store.MarkCompleted(
		context.Background(),
		1,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}

func TestImportJobStoreMarkCompletedNotFound(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create database mock: %v", err)
	}
	defer db.Close()

	store := NewImportJobStore(db)

	db.ExpectExec(`WITH completed_job AS .* UPDATE recaps`).
		WithArgs(
			int64(999),
			string(recap.ImportJobStatusCompleted),
			string(recap.ImportJobStatusProcessing),
		).
		WillReturnResult(
			pgxmock.NewResult("UPDATE", 0),
		)

	err = store.MarkCompleted(
		context.Background(),
		999,
	)

	if !errors.Is(err, recap.ErrImportJobNotFound) {
		t.Errorf(
			"expected ErrImportJobNotFound, got %v",
			err,
		)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet database expectations: %v", err)
	}
}
