package postgres

import (
	"context"
	"errors"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Query(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgx.Rows, error)
	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
	Exec(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgconn.CommandTag, error)
}

type RecapStore struct {
	db DB
}

var _ recap.Store = (*RecapStore)(nil)

func NewRecapStore(db DB) *RecapStore {
	return &RecapStore{
		db: db,
	}
}

func (store *RecapStore) List(
	ctx context.Context,
) ([]recap.Recap, error) {
	rows, err := store.db.Query(ctx, `
		SELECT
			id,
			name,
			status,
			bank_name,
			period,
			created_at,
			updated_at,
			deleted_at
		FROM recaps
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recaps := make([]recap.Recap, 0)

	for rows.Next() {
		var result recap.Recap

		err := rows.Scan(
			&result.ID,
			&result.Name,
			&result.Status,
			&result.BankName,
			&result.Period,
			&result.CreatedAt,
			&result.UpdatedAt,
			&result.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		recaps = append(recaps, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return recaps, nil
}

func (store *RecapStore) GetByID(
	ctx context.Context,
	id int64,
) (recap.Recap, error) {
	var result recap.Recap

	err := store.db.QueryRow(ctx, `
		SELECT
			id,
			name,
			status,
			bank_name,
			period,
			created_at,
			updated_at,
			deleted_at
		FROM recaps
		WHERE id = $1
			AND deleted_at IS NULL
	`, id).Scan(
		&result.ID,
		&result.Name,
		&result.Status,
		&result.BankName,
		&result.Period,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.DeletedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return recap.Recap{}, recap.ErrNotFound
	}

	if err != nil {
		return recap.Recap{}, err
	}

	return result, nil
}

func (store *RecapStore) ListItemsByRecapID(
	ctx context.Context,
	recapID int64,
) ([]recap.Item, error) {
	rows, err := store.db.Query(ctx,
		`
			SELECT
				id,
				recap_id,
				transaction_date::text,
				description,
				amount,
				balance,
				created_at,
				category
			FROM recap_items
			WHERE recap_id = $1
			ORDER BY transaction_date, id
		`, recapID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]recap.Item, 0)

	for rows.Next() {
		var item recap.Item
		var balance *float64
		var category *string

		err := rows.Scan(
			&item.ID,
			&item.RecapID,
			&item.Date,
			&item.Description,
			&item.Amount,
			&balance,
			&item.CreatedAt,
			&category,
		)
		if err != nil {
			return nil, err
		}

		item.Balance = balance
		item.Category = category
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (store *RecapStore) Create(ctx context.Context, input recap.CreateInput) (recap.Recap, error) {
	result := recap.Recap{}

	err := store.db.QueryRow(ctx, `
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
		&result.ID,
		&result.Name,
		&result.Status,
		&result.BankName,
		&result.Period,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.DeletedAt,
	)

	if err != nil {
		return recap.Recap{}, err
	}

	return result, nil
}
