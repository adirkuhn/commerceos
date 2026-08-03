export interface TenantSettings {
  tenant_id: string
  default_currency: string
  timezone: string
  date_format: string
  language: string
  updated_at: string
}

export interface TenantSettingsInput {
  default_currency?: string
  timezone?: string
  date_format?: string
  language?: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
