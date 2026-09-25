import { PickingSessionLineItem, StockShortfall } from './picking'

// ── Packing ──────────────────────────────────────────────────────────────────
// Where an order is scanned, adjusted (substitutes, extra items, refunds)
// and labelled -- picking is optional. See backend domain/oms/packing.go.

export type PackingSessionStatus = 'in_progress' | 'completed'

export interface PackingSession {
  id: string
  sales_order_id: string
  status: PackingSessionStatus
  started_at: string
  completed_at?: string | null
  completed_by: string
  locked_by?: string | null
  locked_at?: string | null
  locked_since?: string | null
}

export type PackScanActionType = 'pack' | 'substitute' | 'freebie' | 'new_item' | 'refund'

// A packing row: 'pack' records box contents; the rest are order changes
// staged until the session completes (see backend PackScan). Field meanings
// for changes mirror PickScan's.
export interface PackScan {
  id: string
  session_id: string
  // Null for a freebie/new item -- it becomes a line only on completion.
  line_item_id: string | null
  sku_id: string
  action_type: PackScanActionType
  quantity: number
  original_quantity_covered?: number | null
  return_to_stock?: boolean
  price_minor_units?: number | null
  reason?: string
  scanned_at: string
  scanned_by: string
  replayed_at?: string | null
}

export interface PackingSessionDetail {
  session: PackingSession
  // Reuses the same loosely-typed line item shape as PickingSessionDetail.
  line_items: PickingSessionLineItem[]
  scans: PackScan[]
}

export interface ScanPackItemInput {
  // Exactly one of code/sku_id is required.
  code?: string
  sku_id?: string
  quantity?: number
  // Defaults to 'pack'. freebie/new_item add an extra item (any SKU) --
  // new_item needs price_minor_units; reason is its name/comment.
  action_type?: 'pack' | 'freebie' | 'new_item'
  price_minor_units?: number
  reason?: string
}

export interface ScanPackItemResult {
  scan: PackScan
}

export interface MissingPackItem {
  line_item_id: string
  name: string
  expected_qty: number
  packed_qty: number
}

// completed:false means nothing was applied or completed -- stock_shortfalls
// lists what needs a decision (resolutions) before finishing again.
export interface CompletePackingResult {
  completed: boolean
  session?: PackingSession
  missing_items?: MissingPackItem[]
  stock_shortfalls?: StockShortfall[]
}
