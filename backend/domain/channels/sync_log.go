package channels

import "time"

type SyncLogStatus string

const (
	SyncLogStatusSuccess SyncLogStatus = "success"
	SyncLogStatusError   SyncLogStatus = "error"
)

type SyncDirectionFlow string

const (
	// SyncFlowPush is CommerceOS -> channel (manual push, event-driven update).
	SyncFlowPush SyncDirectionFlow = "push"
	// SyncFlowImport is channel -> CommerceOS.
	SyncFlowImport SyncDirectionFlow = "import"
)

// FieldChange is one field's before/after value in a SyncLogEntry's Changes map.
type FieldChange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// SyncLogEntry is an immutable audit record of one sync attempt.
// ProductChannelListing holds only the *current* state; this is the
// history of how it got there — created on every push/import/event-driven
// sync, its own fields never updated after creation. ListingID goes nil
// (ON DELETE SET NULL) if the listing it logged activity for is later
// deleted; the log row itself survives as the audit trail. The one
// exception is DismissedAt, set later by staff acknowledging a failure
// isn't worth retrying (see SyncLogRepository.Dismiss) — that's metadata
// about how the entry was handled, not a change to what actually happened.
type SyncLogEntry struct {
	ID           string
	TenantID     string
	ListingID    *string // ProductChannelListing.ID
	ProductID    string
	ConnectionID string
	Provider     string
	ExternalID   string
	Direction    SyncDirectionFlow
	Status       SyncLogStatus
	ErrorMessage string
	// Fields is the full snapshot of synced field values at this point in
	// time (e.g. "name", "price", "status") — what is currently synced.
	Fields map[string]string
	// Changes is the diff against the previous log entry's Fields for this
	// same listing — every field shows as changed from "" on the
	// first-ever sync (nothing existed before), and is empty only when
	// nothing changed since the previous entry.
	Changes  map[string]FieldChange
	SyncedAt time.Time
	// DismissedAt is set once staff acknowledge a failed entry from the
	// "needs attention" view without retrying it (Woo product deleted,
	// known/expected mismatch, etc.) -- see
	// SyncLogRepository.ListNeedingAttention's doc for how this excludes it.
	DismissedAt *time.Time
}
