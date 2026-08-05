package main

import "context"

type RecapItem struct {
	ID          int64   `json:"id"`
	RecapID     int64   `json:"recap_id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      int64   `json:"amount"`
	Balance     *int64  `json:"balance"`
	Category    *string `json:"category"`
}

func (store *postgresRecapStore) ListItemsByRecapID(
	ctx context.Context,
	recapID int64,
) ([]RecapItem, error) {
	rows, err := store.db.Query(ctx,
		`
			SELECT
				id,
				recap_id,
				transaction_date::text,
				description,
				amount,
				balance,
				category
			FROM recap_items
			WHERE recap_id = $1
			ORDER BY transaction_date, id
		`, recapID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RecapItem, 0)

	for rows.Next() {
		var item RecapItem
		err := rows.Scan(
			&item.ID,
			&item.RecapID,
			&item.Date,
			&item.Description,
			&item.Amount,
			&item.Balance,
			&item.Category,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
