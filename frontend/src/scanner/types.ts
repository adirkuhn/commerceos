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

export interface BookingScan {
  id: string
  tenant_id: string
  booking_id: string
  sku_id: string
  batch_id?: string | null
  quantity: number
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
  line_items: Array<{ id: string; sku_id?: string | null; raw_description: string; quantity: number }>
  scans: BookingScan[]
}

export interface ScanBookingItemInput {
  code: string
  quantity: number
  batch_id?: string | null
  lot_number?: string
  expiry_date?: string | null
}

export interface ScanBookingItemResult {
  batch?: Batch | null
  scan: BookingScan
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
