package procurement

import "time"

type DocumentParsingJobStatus string

const (
	DocumentParsingJobPending   DocumentParsingJobStatus = "pending"
	DocumentParsingJobRunning   DocumentParsingJobStatus = "running"
	DocumentParsingJobCompleted DocumentParsingJobStatus = "completed"
	DocumentParsingJobFailed    DocumentParsingJobStatus = "failed"
)

// DocumentParsingJob tracks async extraction+matching of a
// PurchaseOrderDocument's line items, same pending/running/completed/failed
// shape as channels.ImportJob -- picked up and run by cmd/scheduler.
type DocumentParsingJob struct {
	ID              string
	TenantID        string
	PurchaseOrderID string
	DocumentID      string
	Status          DocumentParsingJobStatus
	ErrorMessage    string
	CreatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
	// RetryCount tracks automatic requeue-on-restart attempts -- see
	// DocumentParsingJobRepository.RecoverOrphanedRunning.
	RetryCount int
}
