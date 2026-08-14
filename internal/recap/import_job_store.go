package recap

import "context"

type ImportJobStore interface {
	ClaimNext(ctx context.Context) (*ImportJob, error)

	MarkCompleted(
		ctx context.Context,
		jobID int64,
	) error

	MarkFailed(
		ctx context.Context,
		jobID int64,
		message string,
	) error
}
