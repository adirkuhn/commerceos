export interface TenantSettings {
  tenant_id: string
  default_currency: string
  timezone: string
  date_format: string
  language: string
  prices_include_tax: boolean
  default_tax_country: string
  updated_at: string
}

export interface TenantSettingsInput {
  default_currency?: string
  timezone?: string
  date_format?: string
  language?: string
  prices_include_tax?: boolean
  default_tax_country?: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
