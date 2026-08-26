export interface APIResponse<T> {
  data: T
  error: { code: string; message: string } | null
}

// MyParcel carrier ids -- see backend domain/shipping.Carrier.
export const CARRIERS = [
  { id: 1, label: 'PostNL' },
  { id: 2, label: 'bpost' },
  { id: 4, label: 'DPD' },
  { id: 9, label: 'DHL for you' },
  { id: 10, label: 'DHL Parcel Connect' },
  { id: 11, label: 'DHL Europlus' },
  { id: 12, label: 'UPS Standard' },
  { id: 13, label: 'UPS Express Saver' },
] as const

export const PACKAGE_TYPES = [
  { id: 1, label: 'Package' },
  { id: 2, label: 'Mailbox package' },
  { id: 4, label: 'Digital stamp' },
] as const

export interface RecipientAddress {
  person: string
  company?: string
  street: string
  number?: string
  number_suffix?: string
  postal_code?: string
  city: string
  cc: string
  email?: string
  phone?: string
}

export interface Shipment {
  id: string
  sales_order_id: string
  connection_id: string
  carrier: number
  barcode: string
  status: number
  status_main: string
  label_format: string
  recipient: RecipientAddress
  created_at: string
}

export interface ShippingConnectionSettings {
  connection_id: string
  default_carrier: number
  default_package_type: number
  default_label_format: 'pdf' | 'zpl'
  default_signature_required: boolean
  default_only_recipient: boolean
  default_insurance_amount_minor_units: number | null
  // MyParcel's own merchant account/shop identifiers -- required by their
  // Orders/Fulfilment API export step, not secrets (unlike the API key).
  myparcel_account_id: number | null
  myparcel_shop_id: number | null
  // Auto-opens the print view once a label exists, instead of requiring a
  // manual click -- not silent printing, the OS print dialog still shows;
  // see MyParcelSyncSettingsPage's doc for why.
  auto_print_enabled: boolean
}

export interface ShippingCarrierOverride {
  connection_id: string
  carrier: number
  package_type: number | null
  signature_required: boolean | null
  only_recipient: boolean | null
  insurance_amount_minor_units: number | null
}

export interface ShippingSettingsResult {
  settings: ShippingConnectionSettings | null
  carrier_overrides: ShippingCarrierOverride[]
}

export interface CreateShipmentInput {
  sales_order_id: string
  connection_id: string
  carrier: number
  recipient: RecipientAddress
  package_type?: number | null
  signature_required?: boolean | null
  only_recipient?: boolean | null
  label_format?: 'pdf' | 'zpl' | null
}
