package settings

import "context"

// Repository is scoped to the current tenant via RLS (set by
// tenant.WithTenant) — no tenant ID parameter is needed on either method.
type Repository interface {
	Get(ctx context.Context) (*TenantSettings, error)
	Upsert(ctx context.Context, s *TenantSettings) error

	// GetStorefrontKeyHash/SetStorefrontKeyHash manage the tenant-wide
	// storefront API key (see application/integrations.
	// SetStorefrontKeyUseCase) -- kept off the TenantSettings struct
	// itself so an ordinary settings read (currency, timezone, ...) never
	// carries a credential hash along for the ride. "" means no key has
	// been generated yet.
	GetStorefrontKeyHash(ctx context.Context) (string, error)
	SetStorefrontKeyHash(ctx context.Context, hash string) error
	// FindTenantIDByStorefrontKeyHash resolves a key hash to the tenant it
	// belongs to, across every tenant -- the caller has no tenant to scope
	// by yet, that's exactly what a successful resolution establishes (see
	// application/integrations.ResolveStorefrontKeyUseCase). Returns
	// ErrStorefrontKeyNotFound on no match.
	FindTenantIDByStorefrontKeyHash(ctx context.Context, hash string) (string, error)
}
