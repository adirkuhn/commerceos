package events

// VariantUnlisted is raised when a variant can no longer be sold on the
// channels it was listed on -- the variant was deleted, or its last SKU
// was. Its listing rows are already gone by the time a consumer runs (a
// variant delete cascades them; a last-SKU delete removes them in the same
// transaction), so Listings is a snapshot taken just before, carrying what
// a consumer needs to take the variant off each external channel too --
// same contract as ProductDeleted, one variant instead of a whole product.
type VariantUnlisted struct {
	Base
	TenantID  string                   `json:"tenant_id"`
	ProductID string                   `json:"product_id"`
	VariantID string                   `json:"variant_id"`
	Listings  []VariantUnlistedListing `json:"listings"`
}

type VariantUnlistedListing struct {
	ConnectionID string `json:"connection_id"`
	Provider     string `json:"provider"`
	ExternalID   string `json:"external_id"`
	// ParentExternalID is set only for a Woo variation: the parent
	// variable product's id, needed to address the variation.
	ParentExternalID string `json:"parent_external_id,omitempty"`
}

func (VariantUnlisted) EventName() string { return "pms.variant.unlisted" }
