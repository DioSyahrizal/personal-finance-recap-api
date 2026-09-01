package importer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/diosyahrizal/finance-recap-api/internal/recap"
)

type stubImportJobStore struct {
	job                *recap.ImportJob
	claimErr           error
	claimCalls         int
	completedJobID     int64
	markCompletedCalls int
	markCompletedErr   error
	failedJobID        int64
	failedMessage      string
	markFailedCalls    int
	markFailedErr      error
	claimSignal        chan struct{}
}

func (store *stubImportJobStore) ClaimNext(
	ctx context.Context,
) (*recap.ImportJob, error) {
	store.claimCalls++

	if store.claimSignal != nil {
		select {
		case store.claimSignal <- struct{}{}:
		default:
		}
	}

	return store.job, store.claimErr
}

func (store *stubImportJobStore) MarkCompleted(
	ctx context.Context,
	jobID int64,
) error {
	store.markCompletedCalls++
	store.completedJobID = jobID

	return store.markCompletedErr
}

func (store *stubImportJobStore) MarkFailed(
	ctx context.Context,
	jobID int64,
	message string,
) error {
	store.markFailedCalls++
	store.failedJobID = jobID
	store.failedMessage = message

	return store.markFailedErr
}

type stubProcessor struct {
	calls   int
	lastJob recap.ImportJob
	err     error
}

func (processor *stubProcessor) Process(
	ctx context.Context,
	job recap.ImportJob,
) error {
	processor.calls++
	processor.lastJob = job

	return processor.err
}

func TestWorkerProcessNext(t *testing.T) {
	job := &recap.ImportJob{
		ID:       5,
		RecapID:  12,
		FilePath: "/uploads/statement.pdf",
		Status:   recap.ImportJobStatusProcessing,
		Attempts: 1,
	}

	store := &stubImportJobStore{
		job: job,
	}

	processor := &stubProcessor{}

	worker := NewWorker(
		store,
		processor,
		time.Second,
	)

	err := worker.processNext(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if store.claimCalls != 1 {
		t.Errorf(
			"expected ClaimNext to be called once, got %d",
			store.claimCalls,
		)
	}

	if processor.calls != 1 {
		t.Errorf(
			"expected processor to be called once, got %d",
			processor.calls,
		)
	}

	if processor.lastJob.ID != 5 {
		t.Errorf(
			"expected processor to receive job ID %d, got %d",
			5,
			processor.lastJob.ID,
		)
	}

	if store.markCompletedCalls != 1 {
		t.Errorf(
			"expected MarkCompleted to be called once, got %d",
			store.markCompletedCalls,
		)
	}

	if store.completedJobID != 5 {
		t.Errorf(
			"expected completed job ID %d, got %d",
			5,
			store.completedJobID,
		)
	}

	if store.markFailedCalls != 0 {
		t.Errorf(
			"expected MarkFailed not to be called, got %d",
			store.markFailedCalls,
		)
	}
}

func TestWorkerProcessNextProcessorError(t *testing.T) {
	job := &recap.ImportJob{
		ID:       5,
		RecapID:  12,
		FilePath: "/uploads/statement.pdf",
		Status:   recap.ImportJobStatusProcessing,
		Attempts: 1,
	}

	processErr := errors.New("failed to parse PDF")

	store := &stubImportJobStore{
		job: job,
	}

	processor := &stubProcessor{
		err: processErr,
	}

	worker := NewWorker(
		store,
		processor,
		time.Second,
	)

	err := worker.processNext(context.Background())

	if !errors.Is(err, processErr) {
		t.Errorf(
			"expected processing error, got %v",
			err,
		)
	}

	if processor.calls != 1 {
		t.Errorf(
			"expected processor to be called once, got %d",
			processor.calls,
		)
	}

	if store.markFailedCalls != 1 {
		t.Errorf(
			"expected MarkFailed to be called once, got %d",
			store.markFailedCalls,
		)
	}

	if store.failedJobID != 5 {
		t.Errorf(
			"expected failed job ID %d, got %d",
			5,
			store.failedJobID,
		)
	}

	if store.failedMessage != processErr.Error() {
		t.Errorf(
			"expected failure message %q, got %q",
			processErr.Error(),
			store.failedMessage,
		)
	}

	if store.markCompletedCalls != 0 {
		t.Errorf(
			"expected MarkCompleted not to be called, got %d",
			store.markCompletedCalls,
		)
	}
}

func TestWorkerProcessNextEmptyQueue(t *testing.T) {
	store := &stubImportJobStore{
		job: nil,
	}

	processor := &stubProcessor{}

	worker := NewWorker(
		store,
		processor,
		time.Second,
	)

	err := worker.processNext(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if store.claimCalls != 1 {
		t.Errorf(
			"expected ClaimNext to be called once, got %d",
			store.claimCalls,
		)
	}

	if processor.calls != 0 {
		t.Errorf(
			"expected processor not to be called, got %d",
			processor.calls,
		)
	}

	if store.markCompletedCalls != 0 {
		t.Errorf(
			"expected MarkCompleted not to be called, got %d",
			store.markCompletedCalls,
		)
	}

	if store.markFailedCalls != 0 {
		t.Errorf(
			"expected MarkFailed not to be called, got %d",
			store.markFailedCalls,
		)
	}

}

func TestWorkerRunStopsWhenContextCanceled(t *testing.T) {
	claimSignal := make(chan struct{}, 1)

	store := &stubImportJobStore{
		job:         nil,
		claimSignal: claimSignal,
	}

	processor := &stubProcessor{}

	worker := NewWorker(
		store,
		processor,
		5*time.Millisecond,
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	done := make(chan struct{})

	go func() {
		defer close(done)
		worker.Run(ctx)
	}()

	select {
	case <-claimSignal:
		// The worker ticked and checked the queue.

	case <-time.After(time.Second):
		t.Fatal("worker did not check the queue")
	}

	cancel()

	select {
	case <-done:
		// The worker observed cancellation and exited.

	case <-time.After(time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}

	if store.claimCalls == 0 {
		t.Error("expected worker to claim at least once")
	}
}
