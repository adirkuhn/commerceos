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
