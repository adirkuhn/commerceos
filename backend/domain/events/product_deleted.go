package events

// ProductDeleted is raised after a product has been deleted. Its channel
// listings cascade-delete in the same DB statement (see
// migrations/007_channels.sql, 037_sales_channels.sql's ON DELETE CASCADE),
// so by the time a consumer processes this asynchronously the
// product_channel_listings rows are already gone — Listings is a snapshot
// taken immediately before the delete, carrying what's needed to also
// remove the product from each external channel (e.g. WooCommerce) it was
// actually synced to.
type ProductDeleted struct {
	Base
	TenantID  string                  `json:"tenant_id"`
	ProductID string                  `json:"product_id"`
	Listings  []ProductDeletedListing `json:"listings"`
}

type ProductDeletedListing struct {
	ConnectionID string `json:"connection_id"`
	Provider     string `json:"provider"`
	ExternalID   string `json:"external_id"`
}

func (ProductDeleted) EventName() string { return "pms.product.deleted" }
