import i18next from 'i18next'

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

// id 3 ("letter") is deprecated/unused by MyParcel -- intentionally
// skipped here, same as backend domain/shipping.PackageType.
// Labels are getters so they're read in the current admin language.
export const PACKAGE_TYPES: readonly { readonly id: number; readonly label: string }[] = [
  { id: 1, get label() { return i18next.t('shipping:packageType.package') } },
  { id: 2, get label() { return i18next.t('shipping:packageType.mailbox') } },
  { id: 4, get label() { return i18next.t('shipping:packageType.digitalStamp') } },
  { id: 5, get label() { return i18next.t('shipping:packageType.pallet') } },
  { id: 6, get label() { return i18next.t('shipping:packageType.small') } },
  { id: 7, get label() { return i18next.t('shipping:packageType.envelope') } },
]

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
  // Best available carrier tracking link -- the connection's configured
  // tracking_url_template for this carrier if set, else MyParcel's own
  // link_consumer_portal, else empty (no barcode yet / carrier hasn't
  // offered one).
  tracking_url?: string
  label_format: string
  recipient: RecipientAddress
  created_at: string
  // Set once cancelShipment has been called for this shipment -- see
  // cancel_note for the outcome recorded at cancel time (whether MyParcel
  // itself deleted it or rejected the request).
  cancelled_at?: string | null
  // English; prefer the structured outcome below when set (see
  // cancelOutcomeText).
  cancel_note?: string
  cancel_deleted_on_carrier?: boolean | null
  cancel_carrier_error?: string
  // The carrier kept the pre-announced label (PostNL) -- see
  // CancelShipmentResult.label_kept.
  cancel_label_kept?: boolean
}

// One status MyParcel has reported for a shipment -- the tracking
// timeline. See backend/domain/shipping.ShipmentStatusEvent.
export interface ShipmentTrackingEvent {
  id: string
  shipment_id: string
  status: number
  status_main: string
  status_label: string
  source: 'webhook' | 'reconciliation' | 'manual'
  occurred_at: string
}

// MyParcelFulfilmentOrder is one order exported via MyParcel's real
// Orders/Fulfilment API -- step one of the Sales Order detail page's
// "Create label" flow, before createShipment (the Shipments API) creates
// the actual shipment.
export interface MyParcelFulfilmentOrder {
  id: string
  connection_id: string
  sales_order_id: string
  myparcel_uuid: string
  status: string
}

export interface ShippingConnectionSettings {
  connection_id: string
  default_carrier: number
  default_package_type: number
  default_label_format: 'pdf' | 'zpl'
  // Physical page format for label printing (A4 sheet vs a single A6
  // label) -- a retrieval-time default only, re-askable on every print;
  // see api.ts's getShipmentLabel size parameter.
  default_label_size?: 'A4' | 'A6'
  default_signature_required: boolean
  default_only_recipient: boolean
  // Insurance is on/off only -- the insured amount is never configured
  // here, the backend computes it at shipment-creation time from that
  // specific sales order's own value.
  insure_by_default: boolean
  // MyParcel's own merchant account/shop identifiers -- required by their
  // Orders/Fulfilment API export step, not secrets (unlike the API key).
  myparcel_account_id: number | null
  myparcel_shop_id: number | null
  // Auto-opens the print view once a label exists, instead of requiring a
  // manual click -- not silent printing, the OS print dialog still shows;
  // see MyParcelSyncSettingsPage's doc for why.
  auto_print_enabled: boolean
  // When true, creating a shipment also registers the order via MyParcel's
  // separate Orders/Fulfilment API, so it shows up on MyParcel's own
  // Orders dashboard -- the plain Shipments API used by default does not.
  // Off by default: it hands MyParcel the full order (line items, prices).
  export_order_to_myparcel?: boolean
}

export interface ShippingCarrierOverride {
  connection_id: string
  carrier: number
  package_type: number | null
  signature_required: boolean | null
  only_recipient: boolean | null
  insure_by_default: boolean | null
  // Tracking-page URL template with {barcode}/{postcode} placeholders --
  // e.g. PostNL's "https://postnl.nl/tracktrace/?B={barcode}&P={postcode}&D=NL&T=C&L=EN"
  // or DPD's "https://www.dpdgroup.com/nl/mydpd/my-parcels/search?lang=en&parcelNumber={barcode}".
  // Preferred over MyParcel's own tracking link when set. Null means "use
  // MyParcel's own link, if any".
  tracking_url_template: string | null
}

export interface ShippingSettingsResult {
  settings: ShippingConnectionSettings | null
  carrier_overrides: ShippingCarrierOverride[]
}

// One priced delivery slot from MyParcel's own GET /delivery_options --
// AddressConfirmModal's pre-confirm cost preview, only ever populated for
// NL/BE addresses (MyParcel's own API limitation, see backend's
// domain/shipping.ErrDeliveryEstimateUnsupportedCountry).
export interface DeliveryPriceOption {
  price_comment: string
  amount_minor_units: number
  currency: string
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

// One connection packing can create a label through (MyParcel, PostNL,
// ...) -- see backend GET /shipping/connections.
export interface ShippingConnection {
  id: string
  provider: string
  status: string
  created_at: string
  // Whether packing preselects this connection.
  is_default: boolean
  // What the tenant still has to set up before this connection can create
  // labels; empty = ready. See connectionReadiness.ts.
  missing: MissingSetup[]
}

// Setup a connection can still be missing (backend ShippingConnection.missing).
export const MISSING_SETUP = {
  // The carrier needs a sender and the tenant has neither a shipping
  // sender address nor a company profile address.
  senderAddress: 'sender_address',
} as const
export type MissingSetup = typeof MISSING_SETUP[keyof typeof MISSING_SETUP]

export interface ShippingConnections {
  // Oldest first.
  connections: ShippingConnection[]
  // The connection packing preselects: the merchant's pick, else the
  // oldest one; null only when there is no shipping connection.
  default_connection_id: string | null
  // The merchant's explicit pick; null means "automatic (oldest)".
  configured_connection_id: string | null
}

// A label sender or return address -- backend ShippingAddress.
export interface ShippingAddress {
  company: string
  street: string
  number: string
  number_suffix?: string
  postal_code: string
  city: string
  // ISO 3166-1 alpha-2.
  cc: string
  email?: string
  phone?: string
}

// GET /shipping/addresses: one sender and one return address per tenant.
export interface ShippingAddresses {
  // null = labels use company_address.
  sender_address: ShippingAddress | null
  // null = undeliverable parcels go back to the sender.
  return_address: ShippingAddress | null
  // The company profile address (read-only here); null when not filled in.
  company_address: ShippingAddress | null
}

export type UpdateShippingAddressesInput = Pick<ShippingAddresses, 'sender_address' | 'return_address'>

// A PostNL connection's account and environment -- never the API key.
export interface PostNLAccount {
  connection_id: string
  customer_code: string
  customer_number: string
  sandbox: boolean
}

export interface UpdatePostNLAccountInput {
  // Empty keeps the stored key.
  api_key: string
  customer_code: string
  customer_number: string
  sandbox: boolean
}
