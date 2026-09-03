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
// this package only ever see ID/Name/Description/CreatedAt/RevokedAt.
type APIKey struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	KeyHash     string
	CreatedAt   time.Time
	// RevokedAt is nil for an active key.
	RevokedAt *time.Time
}

func (k APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}
