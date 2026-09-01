package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
	"github.com/jackc/pgx/v5"
)

type ImportJobStore struct {
	db DB
}

var _ recap.ImportJobStore = (*ImportJobStore)(nil)
var _ recap.ImportCreator = (*ImportJobStore)(nil)

func NewImportJobStore(db DB) *ImportJobStore {
	return &ImportJobStore{
		db: db,
	}
}

func (store *ImportJobStore) ClaimNext(
	ctx context.Context,
) (*recap.ImportJob, error) {
	var job recap.ImportJob

	err := store.db.QueryRow(ctx, `
		UPDATE recap_import_jobs
		SET
			status = $1,
			attempts = attempts + 1,
			started_at = NOW(),
			updated_at = NOW()
		WHERE id = (
			SELECT id
			FROM recap_import_jobs
			WHERE status = $2
			ORDER BY created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING
			id,
			recap_id,
			file_path,
			status,
			attempts,
			last_error,
			created_at,
			updated_at,
			started_at,
			completed_at
	`, string(recap.ImportJobStatusProcessing),
		string(recap.ImportJobStatusQueued),
	).Scan(
		&job.ID,
		&job.RecapID,
		&job.FilePath,
		&job.Status,
		&job.Attempts,
		&job.LastError,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.StartedAt,
		&job.CompletedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (store *ImportJobStore) MarkCompleted(
	ctx context.Context,
	jobID int64,
) error {
	result, err := store.db.Exec(ctx, `
		WITH completed_job AS (
			UPDATE recap_import_jobs
			SET
				status = $2,
				last_error = NULL,
				updated_at = NOW(),
				completed_at = NOW()
			WHERE id = $1
				AND status = $3
			RETURNING recap_id
		)
		UPDATE recaps
		SET
			status = $2,
			updated_at = NOW()
		WHERE id = (
			SELECT recap_id
			FROM completed_job
		)
	`,
		jobID,
		string(recap.ImportJobStatusCompleted),
		string(recap.ImportJobStatusProcessing),
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return recap.ErrImportJobNotFound
	}

	return nil
}

func (store *ImportJobStore) MarkFailed(
	ctx context.Context,
	jobID int64,
	message string,
) error {
	result, err := store.db.Exec(ctx, `
		WITH failed_job AS (
			UPDATE recap_import_jobs
			SET
				status = $2,
				last_error = $3,
				updated_at = NOW(),
				completed_at = NOW()
			WHERE id = $1
				AND status = $4
			RETURNING recap_id
		)
		UPDATE recaps
		SET
			status = $2,
			updated_at = NOW()
		WHERE id = (
			SELECT recap_id
			FROM failed_job
		)
	`,
		jobID,
		string(recap.ImportJobStatusFailed),
		message,
		string(recap.ImportJobStatusProcessing),
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return recap.ErrImportJobNotFound
	}

	return nil
}

func (store *ImportJobStore) CreateImport(
	ctx context.Context,
	input recap.CreateInput,
	filePath string,
) (recap.Recap, error) {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return recap.Recap{}, fmt.Errorf(
			"begin create import transaction: %w",
			err,
		)
	}
	defer tx.Rollback(ctx)

	var created recap.Recap

	err = tx.QueryRow(ctx, `
		INSERT INTO recaps (
			name,
			bank_name,
			period
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			name,
			status,
			bank_name,
			period,
			created_at,
			updated_at,
			deleted_at
	`,
		input.Name,
		input.BankName,
		input.Period,
	).Scan(
		&created.ID,
		&created.Name,
		&created.Status,
		&created.BankName,
		&created.Period,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.DeletedAt,
	)
	if err != nil {
		return recap.Recap{}, fmt.Errorf(
			"insert recap: %w",
			err,
		)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO recap_import_jobs (
			recap_id,
			file_path,
			status
		)
		VALUES ($1, $2, $3)
	`,
		created.ID,
		filePath,
		string(recap.ImportJobStatusQueued),
	)
	if err != nil {
		return recap.Recap{}, fmt.Errorf(
			"insert import job: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return recap.Recap{}, fmt.Errorf(
			"commit create import transaction: %w",
			err,
		)
	}

	return created, nil
}
