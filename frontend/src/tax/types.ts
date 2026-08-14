export interface TaxClass {
  id: string
  name: string
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface CreateTaxClassInput {
  name: string
  is_default?: boolean
}

export interface UpdateTaxClassInput {
  name: string
  is_default?: boolean
}

export interface TaxRate {
  id: string
  tax_class_id: string
  country: string
  // 1 basis point = 0.01%, e.g. 2100 = 21.00% -- matches the backend's
  // integer representation so no float rounding creeps in either side.
  rate_basis_points: number
  created_at: string
  updated_at: string
}

export interface UpsertTaxRateInput {
  tax_class_id: string
  country: string
  rate_basis_points: number
}

export interface ExternalTaxClassLink {
  id: string
  provider: string
  // "" is a legitimate value -- e.g. WooCommerce's own Standard rate.
  external_slug: string
  tax_class_id: string
  created_at: string
  updated_at: string
}

export interface UpsertExternalTaxClassLinkInput {
  provider: string
  external_slug: string
  tax_class_id: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
