package settings

import "context"

// Repository is scoped to the current tenant via RLS (set by
// tenant.WithTenant) — no tenant ID parameter is needed on either method.
type Repository interface {
	Get(ctx context.Context) (*TenantSettings, error)
	Upsert(ctx context.Context, s *TenantSettings) error
}
