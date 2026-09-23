// Mirrors backend/api/settings/openapi.yaml (CompanyProfile / CompanyProfileInput).
export interface CompanyAddress {
  street: string
  house_number: string
  postal_code: string
  city: string
  country: string // ISO 3166-1 alpha-2
}

export interface CompanyProfileInput {
  legal_name: string
  trading_name: string
  vat_number: string
  chamber_of_commerce: string
  email: string
  phone: string
  address: CompanyAddress
  iban: string
}

export interface CompanyProfile extends CompanyProfileInput {
  updated_at: string
}

export type CompanyProfileField = 'legal_name' | 'email' | 'vat_number' | 'iban' | 'address.country'
