package moneybird

import (
	"context"
	"time"
)

// SourceType is what a Moneybird document was made from.
type SourceType string

const (
	SourceQuote      SourceType = "quote"
	SourceSalesOrder SourceType = "sales_order"
)

type DocumentStatus string

const (
	DocumentSynced DocumentStatus = "synced"
	// DocumentFailed: the last send failed (Error says why). MoneybirdID is
	// kept when the document itself was created, so a retry resumes it
	// instead of creating a second one.
	DocumentFailed DocumentStatus = "failed"
)

// Document links a quote or sales order to the one Moneybird document it
// was sent as -- an estimate for a quote, an external sales invoice for an
// order.
type Document struct {
	ID                string
	TenantID          string
	ConnectionID      string
	SourceType        SourceType
	SourceID          string
	MoneybirdID       string
	Reference         string
	Status            DocumentStatus
	PaymentRegistered bool
	Error             string
	SyncedAt          time.Time
}

// DocumentRepository stores Document links -- at most one per source.
type DocumentRepository interface {
	Find(ctx context.Context, sourceType SourceType, sourceID string) (*Document, error)
	// Save inserts or replaces the link for (SourceType, SourceID).
	Save(ctx context.Context, d *Document) error
}

// SettingsRepository stores one Settings row per Moneybird connection.
type SettingsRepository interface {
	Get(ctx context.Context, connectionID string) (*Settings, error)
	Upsert(ctx context.Context, s *Settings) error
}

// ContactRepository remembers which Moneybird contact a customer is.
type ContactRepository interface {
	Find(ctx context.Context, connectionID, customerID string) (string, error)
	Save(ctx context.Context, tenantID, connectionID, customerID, moneybirdContactID string) error
}
