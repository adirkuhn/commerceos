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
export const PACKAGE_TYPES = [
  { id: 1, label: 'Package' },
  { id: 2, label: 'Mailbox package' },
  { id: 4, label: 'Digital stamp' },
  { id: 5, label: 'Pallet' },
  { id: 6, label: 'Small package' },
  { id: 7, label: 'Envelope' },
] as const

// CARRIER_PACKAGE_TYPES mirrors backend domain/shipping's carrierPackageTypes
// -- a carrier id present here only accepts these package_type ids (MyParcel
// rejects the rest with a 422). A carrier absent from this map has no
// confirmed restriction and offers every PACKAGE_TYPES id -- only add
// entries backed by MyParcel's own docs/errors, never guessed.
const CARRIER_PACKAGE_TYPES: Partial<Record<number, readonly number[]>> = {
  // PostNL is MyParcel's own-brand "SendMyParcel" default carrier: only
  // Package (1) and Mailbox package (2) are allowed.
  1: [1, 2],
}

export function packageTypesForCarrier(carrierId: number): typeof PACKAGE_TYPES[number][] {
  const allowed = CARRIER_PACKAGE_TYPES[carrierId]
  if (!allowed) return [...PACKAGE_TYPES]
  return PACKAGE_TYPES.filter(p => allowed.includes(p.id))
}

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
  cancel_note?: string
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
