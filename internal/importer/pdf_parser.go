package importer

import (
	"context"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
)

type PDFParser interface {
	Parse(
		ctx context.Context,
		filePath string,
	) ([]recap.Item, error)
}
