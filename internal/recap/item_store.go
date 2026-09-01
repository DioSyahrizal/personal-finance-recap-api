package recap

import "context"

type ItemStore interface {
	CreateMany(
		ctx context.Context,
		recapID int64,
		items []Item,
	) error
}
