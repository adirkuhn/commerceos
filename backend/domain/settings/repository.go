package settings

import "context"

// Repository is scoped to the current tenant via RLS (set by
// tenant.WithTenant) — no tenant ID parameter is needed on either method.
//
// The single anonymous storefront key this interface used to carry
// (GetStorefrontKeyHash/SetStorefrontKeyHash/FindTenantIDByStorefrontKeyHash)
// was superseded by the multiple-named-keys model in domain/apikeys (see
// migration 130_storefront_api_keys.sql) -- removed here since the
// underlying tenant_settings.storefront_key_hash column that migration
// dropped, but this interface and infrastructure/repositories.
// TenantSettingsRepo's GORM model kept referencing.
type Repository interface {
	Get(ctx context.Context) (*TenantSettings, error)
	Upsert(ctx context.Context, s *TenantSettings) error
}

// ExpiryAlertTenantLister lists the settings of every tenant with the
// stock-close-to-expiry alert on -- across tenants, so callers run it with
// RLS bypassed (the scheduler's super-admin context).
type ExpiryAlertTenantLister interface {
	ListExpiryAlertEnabled(ctx context.Context) ([]TenantSettings, error)
}
