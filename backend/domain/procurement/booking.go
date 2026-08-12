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

// BookingScanActionType classifies what a staged BookingScan row will do to
// real inventory once the booking is finished.
type BookingScanActionType string

const (
	BookingScanActionReceive    BookingScanActionType = "receive"
	BookingScanActionCorrection BookingScanActionType = "correction"
	BookingScanActionDamaged    BookingScanActionType = "damaged"
	BookingScanActionLost       BookingScanActionType = "lost"
)

// BookingScan is a STAGED action from the scanner app -- "scan a barcode,
// pick/create a batch, enter quantity" (receive), a correction for an over-/
// mis-scan, or a damaged/lost write-off. ScanBookingItemUseCase only ever
// writes this row -- it never touches real inventory. Nothing becomes a real
// stock movement, and no new batch actually exists, until
// CompleteBookingUseCase replays every staged row for the booking (in
// ScannedAt order) when the booking finishes. This is deliberate: a booking
// in progress is not yet trustworthy (it can be corrected, paused,
// re-scanned), so nothing should reach real inventory -- or, downstream,
// sync to an external sales channel -- until it's done. See
// CompleteBookingUseCase's doc comment for the replay algorithm.
type BookingScan struct {
	ID        string
	TenantID  string
	BookingID string
	SKUID     string
	// BatchID targets an already-real, existing batch. Nil with LotNumber
	// set means "the batch this receive creates" (or, for a
	// correction/damaged/lost row, "the batch an earlier receive row in
	// this same booking will create" -- resolved by matching SKUID+
	// LotNumber during replay, regardless of which row created it). Nil
	// with LotNumber also empty means general ledger.
	BatchID    *string
	LotNumber  string
	ExpiryDate *time.Time
	ActionType BookingScanActionType
	// Quantity is the net effect on this line's "received" progress: +N
	// for a receive, -N for a correction, always 0 for damaged/lost (the
	// stock did arrive -- see BookingScanActionType doc -- so it stays
	// counted as received; only WriteOffQuantity is affected).
	Quantity int64
	// WriteOffQuantity is >0 only on a damaged/lost row -- the amount
	// written off. Zero for receive/correction rows.
	WriteOffQuantity int64
	// Reason is an optional user comment -- most useful on a correction or
	// a damaged/lost row.
	Reason    string
	ScannedAt time.Time
	ScannedBy string
}
