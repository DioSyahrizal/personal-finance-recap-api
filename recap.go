package main

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRecapNotFound = errors.New("recap not found")

type Recap struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Period   string `json:"period"`
	BankName string `json:"bank_name"`
	Status   string `json:"status"`
}

type recapStore interface {
	List(ctx context.Context) ([]Recap, error)
	GetByID(ctx context.Context, id int64) (Recap, error)
	ListItemsByRecapID(
		ctx context.Context,
		recapID int64,
	) ([]RecapItem, error)
}

type postgresRecapStore struct {
	db *pgxpool.Pool
}

func (store *postgresRecapStore) List(
	ctx context.Context,
) ([]Recap, error) {
	rows, err := store.db.Query(ctx, "SELECT id, name, period::text, bank_name, status FROM recaps WHERE deleted_at IS NULL ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recaps := make([]Recap, 0)

	for rows.Next() {
		var recap Recap
		err := rows.Scan(&recap.ID, &recap.Name, &recap.Period, &recap.BankName, &recap.Status)
		if err != nil {
			return nil, err
		}
		recaps = append(recaps, recap)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return recaps, nil
}

func (store *postgresRecapStore) GetByID(
	ctx context.Context,
	id int64,
) (Recap, error) {
	var recap Recap

	err := store.db.QueryRow(ctx, `
			SELECT
				id,
				name,
				period::text,
				bank_name,
				status
			FROM recaps
			WHERE id = $1
			AND deleted_at IS NULL
		`,
		id).Scan(
		&recap.ID,
		&recap.Name,
		&recap.Period,
		&recap.BankName,
		&recap.Status,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Recap{}, ErrRecapNotFound
	}

	if err != nil {
		return Recap{}, err
	}

	return recap, nil
}
