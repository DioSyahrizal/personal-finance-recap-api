package postgres

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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
		var amount pgtype.Float8
		var balance *float64
		var category *string

		err := rows.Scan(
			&item.ID,
			&item.RecapID,
			&item.Date,
			&item.Description,
			&amount,
			&balance,
			&item.CreatedAt,
			&category,
		)
		if err != nil {
			return nil, err
		}

		if amount.Valid {
			value := amount.Float64
			item.Amount = &value
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

func (store *RecapStore) GetAnalytics(ctx context.Context, filter recap.AnalyticsFilter) (recap.Analytics, error) {

	const query = `
	SELECT
    DATE_TRUNC('month', i.transaction_date)::date AS month,
    TO_CHAR(
        DATE_TRUNC('month', i.transaction_date),
        'YYYY-MM'
    ) AS period,

    COALESCE(
        NULLIF(BTRIM(i.category), ''),
        'Uncategorized'
    ) AS category,

    COALESCE(
        SUM(i.amount) FILTER (WHERE i.amount > 0),
        0
    )::double precision AS income,

    COALESCE(
        SUM(ABS(i.amount)) FILTER (WHERE i.amount < 0),
        0
    )::double precision AS expenses,

    COUNT(*)::bigint AS transaction_count,

    COALESCE(
        SUM(ABS(i.amount)) FILTER (
            WHERE i.amount < 0
              AND COALESCE(NULLIF(BTRIM(i.category), ''), 'Uncategorized')
                  = 'Uncategorized'
        ),
        0
    )::double precision AS uncategorized_expenses

	FROM recaps r
	JOIN recap_items i
		ON i.recap_id = r.id

	WHERE r.deleted_at IS NULL
	AND r.status = 'completed'
	AND i.amount IS NOT NULL

	-- from is inclusive: YYYY-MM
	AND (
		$1 = ''
		OR i.transaction_date >= TO_DATE($1, 'YYYY-MM')
	)

	-- to is inclusive: YYYY-MM
	AND (
		$2 = ''
		OR i.transaction_date <
			(TO_DATE($2, 'YYYY-MM') + INTERVAL '1 month')::date
	)

	-- optional bank filter
	AND (
		$3 = ''
		OR r.bank_name = $3
	)

	GROUP BY
		DATE_TRUNC('month', i.transaction_date),
		COALESCE(
			NULLIF(BTRIM(i.category), ''),
			'Uncategorized'
		)

	ORDER BY
		month ASC,
		category ASC;`

	rows, err := store.db.Query(ctx,
		query, filter.From, filter.To, filter.Bank)
	if err != nil {
		return recap.Analytics{}, err
	}
	defer rows.Close()

	analytics := recap.Analytics{
		Series:         make([]recap.AnalyticsPeriod, 0),
		CategoryTotals: make([]recap.AnalyticsCategoryTotal, 0),
	}
	seriesIndexes := make(map[string]int)
	categoryTotals := make(map[string]float64)

	for rows.Next() {
		var (
			month              time.Time
			period             string
			category           string
			income             float64
			expenses           float64
			transactionCount   int64
			uncategorizedTotal float64
		)

		if err := rows.Scan(
			&month,
			&period,
			&category,
			&income,
			&expenses,
			&transactionCount,
			&uncategorizedTotal,
		); err != nil {
			return recap.Analytics{}, err
		}

		seriesIndex, ok := seriesIndexes[period]
		if !ok {
			seriesIndex = len(analytics.Series)
			seriesIndexes[period] = seriesIndex
			analytics.Series = append(analytics.Series, recap.AnalyticsPeriod{
				Period:     period,
				Categories: make(map[string]float64),
			})
		}

		series := &analytics.Series[seriesIndex]
		series.Income += income
		series.Expenses += expenses
		if category != string(recap.CategoryIncome) && expenses > 0 {
			series.Categories[category] += expenses
		}

		analytics.Summary.TotalIncome += income
		analytics.Summary.TotalExpenses += expenses
		analytics.Summary.TransactionCount += int(transactionCount)
		analytics.Summary.UncategorizedTotal += uncategorizedTotal

		if category != string(recap.CategoryIncome) && expenses > 0 {
			categoryTotals[category] += expenses
		}
	}

	if err := rows.Err(); err != nil {
		return recap.Analytics{}, err
	}

	for index := range analytics.Series {
		analytics.Series[index].NetChange =
			analytics.Series[index].Income - analytics.Series[index].Expenses
	}
	analytics.Summary.NetChange =
		analytics.Summary.TotalIncome - analytics.Summary.TotalExpenses

	categoryNames := make([]string, 0, len(categoryTotals))
	for category := range categoryTotals {
		categoryNames = append(categoryNames, category)
	}
	sort.Strings(categoryNames)

	for _, category := range categoryNames {
		total := categoryTotals[category]
		percentage := 0.0
		if analytics.Summary.TotalExpenses > 0 {
			percentage = total / analytics.Summary.TotalExpenses * 100
		}

		analytics.CategoryTotals = append(
			analytics.CategoryTotals,
			recap.AnalyticsCategoryTotal{
				Category:   category,
				Total:      total,
				Percentage: percentage,
			},
		)
	}

	return analytics, nil
}
