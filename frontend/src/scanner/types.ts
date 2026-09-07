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

export type PickingSessionStatus = 'in_progress' | 'paused' | 'completed'

export interface PickingSession {
  id: string
  sales_order_id: string
  status: PickingSessionStatus
  started_at: string
  paused_at?: string | null
  completed_at?: string | null
  completed_by: string
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
  // this row covers. Every substitute row for a given original line,
  // summed, equals that line's quantity exactly. Null for every other
  // action type.
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
  // Defaults to 'pick' server-side when omitted. 'substitute' is rejected
  // (422) -- use substituteLineItem instead.
  action_type?: PickScanActionType
  // Unused -- substitute (the only action type that ever needed this) is
  // rejected here.
  line_item_id?: string
  quantity?: number
  // Required for new_item, ignored (forced to 0) for freebie.
  price_minor_units?: number
  // Optional comment (pick), or the new line item's name (freebie/new_item).
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

// Stages one or more Substitute PickScan rows for the same original line
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
  scans: PickScan[]
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
  scan: PickScan
}

// Result of resolving a scanned code to a SKU WITHOUT staging anything yet
// -- same "confirm before it's real" split as ResolveBookingScanCodeResult.
export interface ResolvePickScanCodeResult {
  sku_id: string
  product_name: string
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

export interface AddSalesOrderLineItemInput {
  sku_id: string
  name?: string
  quantity: number
  price_minor_units: number
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
