package importer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
)

type Processor interface {
	Process(
		ctx context.Context,
		job recap.ImportJob,
	) error
}

type Worker struct {
	store     recap.ImportJobStore
	processor Processor
	interval  time.Duration
}

const processTimeout = 2 * time.Minute

func NewWorker(
	store recap.ImportJobStore,
	processor Processor,
	interval time.Duration,
) *Worker {
	return &Worker{
		store:     store,
		processor: processor,
		interval:  interval,
	}
}

func (worker *Worker) processNext(
	ctx context.Context,
) error {
	job, err := worker.store.ClaimNext(ctx)
	if err != nil {
		return fmt.Errorf("claim next import job: %w", err)
	}

	if job == nil {
		return nil
	}

	log.Printf(
		"import worker: processing job_id=%d recap_id=%d",
		job.ID,
		job.RecapID,
	)

	processCtx, cancel := context.WithTimeout(ctx, processTimeout)
	err = worker.processor.Process(processCtx, *job)
	cancel()
	if err != nil {
		markCtx := context.WithoutCancel(ctx)
		markErr := worker.store.MarkFailed(
			markCtx,
			job.ID,
			err.Error(),
		)
		if markErr != nil {
			return fmt.Errorf(
				"process import job %d: %v; mark failed: %w",
				job.ID,
				err,
				markErr,
			)
		}

		return fmt.Errorf(
			"process import job %d: %w",
			job.ID,
			err,
		)
	}

	if err := worker.store.MarkCompleted(ctx, job.ID); err != nil {
		return fmt.Errorf(
			"mark import job %d completed: %w",
			job.ID,
			err,
		)
	}

	log.Printf(
		"import worker: completed job_id=%d recap_id=%d",
		job.ID,
		job.RecapID,
	)

	return nil
}

func (worker *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(worker.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if err := worker.processNext(ctx); err != nil {
				log.Printf("import worker: %v", err)
			}
		}

	}
}
