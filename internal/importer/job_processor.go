package importer

import (
	"context"
	"fmt"
	"log"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
)

type JobProcessor struct {
	parser    PDFParser
	itemStore recap.ItemStore
	fileStore FileStore
}

var _ Processor = (*JobProcessor)(nil)

func NewJobProcessor(
	parser PDFParser,
	itemStore recap.ItemStore,
	fileStore FileStore,
) *JobProcessor {
	return &JobProcessor{
		parser:    parser,
		itemStore: itemStore,
		fileStore: fileStore,
	}
}

func (processor *JobProcessor) Process(
	ctx context.Context,
	job recap.ImportJob,
) error {
	items, err := processor.parser.Parse(ctx, job.FilePath)
	if err != nil {
		return fmt.Errorf("parse import PDF: %w", err)
	}

	if len(items) == 0 {
		return fmt.Errorf("parser returned no transactions")
	}

	if err := processor.itemStore.CreateMany(
		ctx,
		job.RecapID,
		items,
	); err != nil {
		return fmt.Errorf("save parsed transactions: %w", err)
	}

	if err := processor.fileStore.Delete(
		context.WithoutCancel(ctx),
		job.FilePath,
	); err != nil {
		// The transaction succeeded, so do not make the job retry and insert
		// duplicate items just because local cleanup failed.
		log.Printf(
			"failed to delete processed PDF %q: %v",
			job.FilePath,
			err,
		)
	}

	return nil
}
