export interface TenantSettings {
  tenant_id: string
  default_currency: string
  timezone: string
  date_format: string
  language: string
  prices_include_tax: boolean
  default_tax_country: string
  inventory_sync_rules_enabled: boolean
  batch_expiry_threshold_days: number
  // Tenant-wide fallback target margin (0-9999 basis points, e.g. 3000 =
  // 30%) used to compute a SKU's recommended price when its category has
  // no override (see pms.Category.target_margin_basis_points). 0 on a
  // never-saved tenant means "unset".
  default_target_margin_basis_points: number
  updated_at: string
}

export interface TenantSettingsInput {
  default_currency?: string
  timezone?: string
  date_format?: string
  language?: string
  prices_include_tax?: boolean
  default_tax_country?: string
  inventory_sync_rules_enabled?: boolean
  batch_expiry_threshold_days?: number
  default_target_margin_basis_points?: number
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
