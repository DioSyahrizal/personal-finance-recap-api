package recap

import (
	"errors"
	"time"
)

type ImportJobStatus string

const (
	ImportJobStatusQueued     ImportJobStatus = "queued"
	ImportJobStatusProcessing ImportJobStatus = "processing"
	ImportJobStatusCompleted  ImportJobStatus = "completed"
	ImportJobStatusFailed     ImportJobStatus = "failed"
)

type ImportJob struct {
	ID          int64
	RecapID     int64
	FilePath    string
	Status      ImportJobStatus
	Attempts    int
	LastError   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
}

var ErrImportJobNotFound = errors.New(
	"import job not found or not processing",
)
