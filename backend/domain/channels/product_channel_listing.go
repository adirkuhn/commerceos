// Package channels holds the cross-cutting concept of a CommerceOS product
// being listed on an external sales channel (WooCommerce today, others
// later). Deliberately its own bounded context rather than living inside
// pms or integrations — it references both by ID only, never by importing
// their domain types, so pms and integrations stay decoupled from each
// other and from any specific channel provider.
package channels

import "time"

type SyncStatus string

const (
	SyncStatusPending SyncStatus = "pending"
	SyncStatusSynced  SyncStatus = "synced"
	SyncStatusError   SyncStatus = "error"
)

type ProductChannelListing struct {
	ID            string
	TenantID      string
	ProductID     string // pms.Product.ID
	ConnectionID  string // integrations.Connection.ID
	Provider      string // denormalized from the connection, e.g. "woocommerce"
	ExternalID    string // the product's id on the channel's own system
	SyncStatus    SyncStatus
	LastSyncedAt  *time.Time
	LastSyncError string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
