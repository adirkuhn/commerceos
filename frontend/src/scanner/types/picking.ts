import { PackScan } from './packing'

export type PickingSessionStatus = 'in_progress' | 'paused' | 'completed'

export interface PickingSession {
  id: string
  sales_order_id: string
  status: PickingSessionStatus
  started_at: string
  paused_at?: string | null
  completed_at?: string | null
  completed_by: string
  locked_by?: string | null
  locked_at?: string | null
  locked_since?: string | null
}

export type PickScanActionType = 'pick' | 'substitute' | 'freebie' | 'new_item' | 'refund'

// A STAGED action -- see backend domain/oms/picking.go's doc comment.
// Nothing here is real (no stock movement, no order line item) until
// CompletePickingUseCase replays every row for the session.
export interface PickScan {
  id: string
  session_id: string
  // The line being picked (pick), the ORIGINAL line being replaced
  // (substitute), or null (freebie/new_item -- nothing on the order yet).
  line_item_id?: string | null
  // The line's own SKU (pick), the SUBSTITUTE sku (substitute), or the new
  // SKU being added (freebie/new_item).
  sku_id: string
  action_type: PickScanActionType
  // Always a positive magnitude -- for substitute, this is the SUBSTITUTE
  // sku's own quantity, not the original line's quantity.
  quantity: number
  // Substitute only -- how much of the original line item's own quantity
  // this row covers. A single substitute batch's rows, summed, must equal
  // the line's REMAINING (not yet covered by any prior Pick/Substitute/
  // Refund scan) quantity exactly -- a partially-picked line can still
  // have the rest substituted. Null for every other action type.
  original_quantity_covered?: number | null
  // Substitute only -- whether this row's own original_quantity_covered
  // was added back to the original SKU's stock at replay. Defaults false.
  return_to_stock?: boolean
  // Null for pick/substitute, 0 for freebie, picker-entered for new_item.
  price_minor_units?: number | null
  // Optional comment on a pick/substitute row -- doubles as the new line
  // item's name for freebie/new_item.
  reason?: string
  scanned_at: string
  scanned_by: string
}

// Loosely typed reuse of oms's SalesOrderLineItem shape, same reasoning as
// BookingDetail's line_items above -- avoids a hard cross-module import
// cycle between scanner and oms.
export interface PickingSessionLineItem {
  id: string
  sku_id?: string | null
  name: string
  sku_code: string
  barcode: string
  quantity: number
  price_minor_units: number
  total_minor_units: number
  // Display-only -- the current product this SKU belongs to, used to fetch
  // a thumbnail (see backend SalesOrderLineItem.ProductID's doc). Null when
  // sku_id is null or the SKU/variant has since been deleted.
  product_id?: string | null
  // What WooCommerce reported as the expiry date sold, snapshotted at
  // import time -- see backend SalesOrderLineItem.SoldExpiryDate's doc.
  // Null for every non-Woo line, or when Woo had no expiry meta set.
  sold_expiry_date?: string | null
}

export interface PickingSessionDetail {
  session: PickingSession
  line_items: PickingSessionLineItem[]
  scans: PickScan[]
}

export interface ScanPickItemInput {
  // Exactly one of code/sku_id is required.
  code?: string
  sku_id?: string
  // Picking only picks -- substitutes, extra items and refunds are
  // recorded in packing.
  action_type?: 'pick'
  quantity?: number
  // Optional comment.
  reason?: string
}

export interface ScanPickItemResult {
  scan: PickScan
}

// One row of a substitute batch -- see SubstituteLineItemInput.
export interface SubstituteLine {
  sku_id: string
  substitute_quantity: number
  // How much of the original line item's own quantity this row covers.
  original_quantity_covered: number
  // Defaults false server-side when omitted -- a substitution normally
  // means the original stock figure was wrong, so nothing is added back
  // unless staff explicitly opt in.
  return_to_stock?: boolean
}

// Stages one or more substitute PackScan rows for the same original line
// item in a single call, one per substitute SKU -- lets a single original
// unit be replaced by several different (typically cheaper) substitutes at
// once. lines' original_quantity_covered must sum to the original line
// item's own quantity exactly.
export interface SubstituteLineItemInput {
  line_item_id: string
  lines: SubstituteLine[]
  reason?: string
}

export interface SubstituteLineItemResult {
  scans: PackScan[]
}

// Stages refunding part or all of an existing line item's own quantity,
// when staff choose not to substitute it -- see PickScanActionRefund's
// backend doc comment. Real money only moves at session completion; a
// customer_user caller needs the oms.line_item.refund permission (403
// otherwise).
export interface RefundLineItemRequest {
  line_item_id: string
  // At most the line item's own quantity.
  quantity: number
  reason?: string
}

export interface RefundLineItemResult {
  scan: PackScan
}

// Result of resolving a scanned code to a SKU WITHOUT staging anything yet
// -- same "confirm before it's real" split as ResolveBookingScanCodeResult.
export interface ResolvePickScanCodeResult {
  sku_id: string
  product_name: string
  // Non-null only when the scanned code was a weight-embedded (butcher/
  // deli scale) barcode -- the actual weight of the specific package
  // scanned. Prefill the pick quantity from this (in kg) instead of 1.
  weight_grams?: number | null
}

export interface MissingPickItem {
  line_item_id: string
  name: string
  expected_qty: number
  picked_qty: number
}

// One substitute/freebie/new-item scan whose SKU doesn't have enough
// tracked stock in the order's warehouse -- returned instead of completing
// (completed:false) until staff resolve it (see StockShortfallResolution).
export interface StockShortfall {
  scan_id: string
  sku_id: string
  name: string
  needed: number
  short: number
}

export type StockShortfallResolution = 'with_item' | 'without_item'

export interface CompletePickingResult {
  // completed:false means nothing was replayed or completed --
  // stock_shortfalls lists what needs a decision before resubmitting.
  // session/missing_items are absent in that case.
  completed: boolean
  session?: PickingSession
  missing_items?: MissingPickItem[]
  stock_shortfalls?: StockShortfall[]
}
