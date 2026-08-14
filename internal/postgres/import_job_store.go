package postgres

import (
	"context"
	"errors"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
	"github.com/jackc/pgx/v5"
)

type ImportJobStore struct {
	db DB
}

var _ recap.ImportJobStore = (*ImportJobStore)(nil)

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
