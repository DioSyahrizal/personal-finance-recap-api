package importer

import (
	"context"
	"io"
)

type FileStore interface {
	Save(
		ctx context.Context,
		source io.Reader,
	) (string, error)

	Delete(
		ctx context.Context,
		path string,
	) error
}
