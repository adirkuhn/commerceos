import { describe, it, expect } from 'vitest'
import { toInput } from '../tenantSettingsInput'
import type { TenantSettings } from '../types'

const settings: TenantSettings = {
  tenant_id: 't', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
  prices_include_tax: true, default_tax_country: 'NL', inventory_sync_rules_enabled: true, batch_expiry_threshold_days: 30,
  default_target_markup_basis_points: 3000, bundle_confidence_threshold: 0.3, recommendations_min_support: 2,
  recommendations_lookback_days: 180, picking_mode: 'list', updated_at: '2026-09-24T00:00:00Z',
}

describe('toInput', () => {
  it('carries every persisted field -- PATCH replaces the whole row', () => {
    expect(toInput(settings)).toEqual({
      default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'NL', inventory_sync_rules_enabled: true, batch_expiry_threshold_days: 30,
      default_target_markup_basis_points: 3000, bundle_confidence_threshold: 0.3, recommendations_min_support: 2,
      recommendations_lookback_days: 180, picking_mode: 'list',
    })
  })

  it('is empty before settings load', () => {
    expect(toInput(null)).toEqual({})
  })
})
