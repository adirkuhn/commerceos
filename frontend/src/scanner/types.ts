export type BookingStatus = 'in_progress' | 'paused' | 'completed'

export interface Booking {
  id: string
  tenant_id: string
  purchase_order_id: string
  status: BookingStatus
  started_at: string
  paused_at?: string | null
  completed_at?: string | null
  completed_by: string
}

export type BookingScanActionType = 'receive' | 'correction' | 'damaged' | 'lost'

// A STAGED action -- see backend domain/procurement/booking.go's doc
// comment. Nothing here is real inventory yet; CompleteBookingUseCase
// replays every row for the booking when it finishes.
export interface BookingScan {
  id: string
  tenant_id: string
  booking_id: string
  sku_id: string
  // An already-real, existing batch. Null with lot_number set means "the
  // batch this row creates or targets" -- not real until the booking
  // finishes.
  batch_id?: string | null
  lot_number?: string
  expiry_date?: string | null
  action_type: BookingScanActionType
  // Net effect on this line's received progress -- 0 for damaged/lost.
  quantity: number
  // The amount written off -- only nonzero on a damaged/lost row.
  write_off_quantity: number
  reason?: string
  scanned_at: string
  scanned_by: string
}

export interface Batch {
  id: string
  tenant_id: string
  sku_id: string
  warehouse_id: string
  lot_number: string
  expiry_date?: string | null
  received_at: string
  created_at: string
  updated_at: string
}

export interface BookingDetail {
  booking: Booking
  // Reuses procurement's PurchaseOrderLineItem shape -- typed loosely here
  // to avoid a hard cross-module import cycle; see ../procurement/types.
  // units_per_purchased_item lets the client mirror
  // PurchaseOrderLineItem.EffectiveUnitsPerPurchasedItem's case-pack scaling
  // (see computeMissingItems in BookingScreen.tsx) for the pre-completion
  // preview -- optional here since older/loose fixtures omit it; treat
  // missing/<=0 as 1, same as the backend.
  // suggested_lot_number/suggested_expiry_date are parsed from the source
  // document (e.g. an invoice printing a lot code + expiry next to the
  // line) -- BookLineSheet pre-fills its lot number/expiry inputs from
  // these when present, but a human still has to hit Confirm before
  // anything real is created.
  line_items: Array<{
    id: string
    sku_id?: string | null
    raw_description: string
    quantity: number
    units_per_purchased_item?: number
    suggested_lot_number?: string
    suggested_expiry_date?: string | null
    // product_id is display-only, set only when sku_id is linked -- lets
    // the booking list fetch a thumbnail via useProductMedia.
    product_id?: string | null
  }>
  scans: BookingScan[]
}

export interface ScanBookingItemInput {
  // Exactly one of code/sku_id is required -- code for a scanned barcode,
  // sku_id to book directly against an already-known SKU (tapping a line
  // item row).
  code?: string
  sku_id?: string
  // Defaults to 'receive' server-side when omitted.
  action_type?: BookingScanActionType
  // Always a positive magnitude -- action_type carries the direction.
  quantity: number
  batch_id?: string | null
  lot_number?: string
  expiry_date?: string | null
  // Optional comment, most useful on a correction or a damaged/lost row.
  reason?: string
}

export interface ScanBookingItemResult {
  scan: BookingScan
}

// Result of resolving a scanned code to a SKU WITHOUT booking anything --
// see BookingScreen.tsx's resolveAndOpenSheet, which uses this to open the
// same confirm screen (quantity, batch) that tapping a line item row does,
// instead of ScanBookingItemInput's old instant-book-on-scan behavior.
export interface ResolveBookingScanCodeResult {
  sku_id: string
  product_name: string
  // 1 for a direct unit-barcode/SKU-code/line-item match, or the case/box
  // multiplier when resolved via a BarcodeAlias -- multiply the scanned
  // quantity by this before defaulting the confirm screen's quantity field,
  // since the eventual book-it call sends sku_id (not code) and so skips
  // the backend's own alias-multiplier step.
  units_per_scan: number
}

export interface MissingBookingItem {
  line_item_id: string
  raw_description: string
  expected_qty: number
  scanned_qty: number
}

export interface CompleteBookingResult {
  booking: Booking
  missing_items: MissingBookingItem[]
}

export type PairingCodeStatus = 'pending' | 'claimed'

export interface ScannerPairingCode {
  id: string
  tenant_id: string
  code: string
  // High-entropy opaque value QR codes / barcodes encode -- unguessable
  // within the pairing TTL, unlike code.
  token: string
  status: PairingCodeStatus
  expires_at: string
  created_at: string
  claimed_at?: string | null
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
