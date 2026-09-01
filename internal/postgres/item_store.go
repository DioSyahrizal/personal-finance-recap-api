package postgres

import (
	"context"
	"fmt"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
)

type ItemStore struct {
	db DB
}

var _ recap.ItemStore = (*ItemStore)(nil)

func NewItemStore(db DB) *ItemStore {
	return &ItemStore{
		db: db,
	}
}

func (store *ItemStore) CreateMany(
	ctx context.Context,
	recapID int64,
	items []recap.Item,
) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := store.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"begin create items transaction: %w",
			err,
		)
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		category := recap.CategoryUncategorized
		if item.Category != nil {
			category = recap.NormalizeCategory(*item.Category)
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO recap_items (
				recap_id,
				transaction_date,
				description,
				amount,
				balance,
				category
			)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
			recapID,
			item.Date,
			item.Description,
			item.Amount,
			item.Balance,
			string(category),
		)
		if err != nil {
			return fmt.Errorf(
				"insert recap item: %w",
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"commit create items transaction: %w",
			err,
		)
	}

	return nil
}
