package recap

import "context"

type Store interface {
	List(ctx context.Context) ([]Recap, error)
	GetByID(ctx context.Context, id int64) (Recap, error)
	ListItemsByRecapID(
		ctx context.Context,
		recapID int64,
	) ([]Item, error)
	Create(
		ctx context.Context,
		input CreateInput,
	) (Recap, error)
	GetAnalytics(
		ctx context.Context,
		filter AnalyticsFilter,
	) (Analytics, error)
}
