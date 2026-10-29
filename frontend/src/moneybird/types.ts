// Mirrors backend/api/moneybird/openapi.yaml.

export interface APIResponse<T> {
  data: T | null
  error: { code: string; message: string; detail?: string } | null
}

export interface TaxRateMapping {
  rate_basis_points: number
  moneybird_tax_rate_id: string
}

export type MoneybirdPaymentMethod = 'cash' | 'credit_card' | 'debit_card' | 'bank_transfer' | 'ideal' | 'paypal' | 'pin'

export interface PaymentMethodMapping {
  method: string
  moneybird_payment_method: MoneybirdPaymentMethod
}

export interface MoneybirdSettings {
  connection_id: string
  administration_id: string
  administration_name: string
  default_ledger_account_id: string
  auto_sync: boolean
  tax_rates: TaxRateMapping[]
  payment_methods: PaymentMethodMapping[]
}

// Only the fields sent change -- each settings section sends its own.
export interface UpdateMoneybirdSettingsInput {
  administration_id?: string
  default_ledger_account_id?: string
  auto_sync?: boolean
  tax_rates?: TaxRateMapping[]
  payment_methods?: PaymentMethodMapping[]
}

export interface MoneybirdAdministration {
  id: string
  name: string
  currency: string
}

export interface MoneybirdTaxRate {
  id: string
  name: string
  percentage: string
}

export interface MoneybirdLedgerAccount {
  id: string
  name: string
}

export interface MoneybirdSetup {
  settings: MoneybirdSettings
  administrations: MoneybirdAdministration[]
  tax_rates: MoneybirdTaxRate[]
  ledger_accounts: MoneybirdLedgerAccount[]
  payment_methods: MoneybirdPaymentMethod[]
}

export type MoneybirdSourceType = 'quote' | 'sales_order'

export interface MoneybirdDocument {
  source_type: MoneybirdSourceType
  source_id: string
  // "" when the last send failed before the document was created.
  moneybird_id: string
  reference: string
  status: 'synced' | 'failed'
  payment_registered: boolean
  error: string
  synced_at: string
}

export interface MoneybirdDocumentState {
  document: MoneybirdDocument | null
  administration_id: string
  auto_sync: boolean
}
