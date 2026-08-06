package procurement

import "time"

type BookingStatus string

const (
	BookingStatusInProgress BookingStatus = "in_progress"
	BookingStatusPaused     BookingStatus = "paused"
	BookingStatusCompleted  BookingStatus = "completed"
)

// Booking tracks one scanner-app session working a purchase order on the
// warehouse floor -- StartOrResumeBookingUseCase reuses an existing
// in_progress/paused booking for the PO rather than creating a new one, so
// two staff scanning the same PO (or one pausing and resuming later) share
// the same running total.
type Booking struct {
	ID              string
	TenantID        string
	PurchaseOrderID string
	Status          BookingStatus
	StartedAt       time.Time
	PausedAt        *time.Time
	CompletedAt     *time.Time
	// CompletedBy is the paired session's UserID (see
	// iam.ScannerPairingCode) -- plain string, not a FK, same convention as
	// every other *By field in this codebase (lm2 user IDs aren't UUIDs).
	CompletedBy string
}

// BookingScan is the audit-trail row for one "scan a barcode, pick/create a
// batch, enter quantity" action from the scanner app. ScanBookingItemUseCase
// composes inventory.ReceiveStockUseCase (which opens its own transaction)
// and then writes this row in a second transaction -- same two-step,
// not-fully-atomic composition CreatePurchaseOrderFromDocumentUseCase uses
// for the same reason: tenant.WithTenant must never nest, so a use case
// that calls another use case's Execute can't wrap both in one transaction.
// Worst case on a mid-sequence crash is stock booked with no audit row,
// which the booking's own progress view can reconcile against the ledger.
type BookingScan struct {
	ID        string
	TenantID  string
	BookingID string
	SKUID     string
	BatchID   *string
	Quantity  int64
	ScannedAt time.Time
	ScannedBy string
}
