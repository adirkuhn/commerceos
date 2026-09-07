// Package apikeys models a tenant's storefront API keys -- what a
// connected storefront (e.g. the WooCommerce plugin) presents as a bearer
// token to call CommerceOS' authenticated public-storefront API. Several
// named, independently revocable keys per tenant, unlike the single
// anonymous rotate-only key this replaced.
package apikeys

import "time"

// APIKey never carries the plaintext key past generation time -- only
// KeyHash persists, and even that stays inside the repository layer (see
// Repository.FindActiveTenantIDByHash); application-layer callers outside
// this package only ever see ID/Name/Description/CreatedAt/RevokedAt/
// SalesChannelID.
type APIKey struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	KeyHash     string
	// SalesChannelID is nil for a tenant-wide key (e.g. the WooCommerce
	// plugin's, or any key minted before this field existed). Set scopes
	// the key to one specific channels.SalesChannel -- what lets a tenant
	// run more than one internal storefront (e.g. B2C and B2B) with each
	// one's checkout key resolving to its own catalog/pricing rules. See
	// application/storefront's use cases, the only callers that read this.
	SalesChannelID *string
	CreatedAt      time.Time
	// RevokedAt is nil for an active key.
	RevokedAt *time.Time
}

func (k APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}

// ResolvedAPIKey is what a plaintext key resolves to -- see
// Repository.FindActiveTenantIDByHash.
type ResolvedAPIKey struct {
	TenantID string
	// SalesChannelID mirrors APIKey.SalesChannelID.
	SalesChannelID *string
}
