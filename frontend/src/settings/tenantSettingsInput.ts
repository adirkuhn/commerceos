import type { TenantSettings, TenantSettingsInput } from './types'

// toInput picks just the persisted fields off a fetched TenantSettings so a
// section that only edits some of them (Localization edits 4, Tax edits 2)
// can still send a full snapshot -- the backend replaces the whole row on
// every PATCH (see UpdateTenantSettingsUseCase), so omitting a field here
// would silently wipe it back to empty/false instead of leaving it alone.
// Every new persisted field must be added here.
export function toInput(s: TenantSettings | null): TenantSettingsInput {
  if (!s) return {}
  return {
    default_currency: s.default_currency, timezone: s.timezone, date_format: s.date_format, language: s.language,
    prices_include_tax: s.prices_include_tax, default_tax_country: s.default_tax_country,
    inventory_sync_rules_enabled: s.inventory_sync_rules_enabled, batch_expiry_threshold_days: s.batch_expiry_threshold_days,
    default_target_markup_basis_points: s.default_target_markup_basis_points,
    bundle_confidence_threshold: s.bundle_confidence_threshold,
    recommendations_min_support: s.recommendations_min_support,
    recommendations_lookback_days: s.recommendations_lookback_days,
    picking_mode: s.picking_mode,
  }
}
