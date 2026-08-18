// Address/contact fields mirror Shopify's Location resource.
export interface Warehouse {
  id: string
  tenant_id: string
  name: string
  address1: string
  address2: string
  city: string
  province: string
  zip: string
  country: string
  phone: string
  active: boolean
  created_at: string
  updated_at: string
}

export interface WarehouseInput {
  name: string
  address1?: string
  address2?: string
  city?: string
  province?: string
  zip?: string
  country?: string
  phone?: string
  active?: boolean
}

export type MovementType = 'receipt' | 'sale' | 'adjustment' | 'transfer_out' | 'transfer_in' | 'damaged' | 'lost' | 'split_out' | 'split_in'

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
  // Computed from the stock movement ledger, never stored -- see the
  // backend's ListBatchesUseCase.
  quantity: number
  // True for a synthesized row representing a warehouse's un-lotted
  // ("general ledger") stock rather than a real batch.
  is_general_ledger: boolean
}

export interface CreateBatchInput {
  warehouse_id: string
  lot_number?: string
  expiry_date?: string | null
  received_at?: string
}

export interface UpdateBatchInput {
  warehouse_id: string
  lot_number?: string
  expiry_date?: string | null
}

export interface StockMovement {
  id: string
  tenant_id: string
  sku_id: string
  warehouse_id: string
  batch_id?: string | null
  type: MovementType
  quantity_delta: number
  reason: string
  occurred_at: string
  created_at: string
}

export interface StockLevel {
  sku_id: string
  total: number
  syncable: number
}

export interface RecordStockMovementInput {
  warehouse_id: string
  batch_id?: string | null
  type: MovementType
  quantity_delta: number
  reason?: string
  occurred_at?: string
}

export interface RecordRecountInput {
  warehouse_id: string
  batch_id?: string | null
  counted_quantity: number
  reason?: string
  occurred_at?: string
}

export interface TransferStockInput {
  quantity: number
  from_warehouse_id: string
  from_batch_id?: string | null
  to_warehouse_id: string
  to_batch_id?: string | null
  reason?: string
  occurred_at?: string
}

export interface TransferStockResult {
  out: StockMovement
  in: StockMovement
}

export interface SplitCaseIntoUnitsInput {
  warehouse_id: string
  batch_id?: string | null
  cases_to_split: number
  reason?: string
  occurred_at?: string
}

export interface SplitCaseIntoUnitsResult {
  case_out: StockMovement
  unit_in: StockMovement
}

export interface PortionFromBulkInput {
  warehouse_id: string
  batch_id?: string | null
  // The actual amount consumed for this one physical item (e.g. a scale
  // reading). Omit to default to the target SKU's own nominal weight_grams.
  weight_grams?: number
  reason?: string
  occurred_at?: string
}

export interface PortionFromBulkResult {
  bulk_out: StockMovement
  unit_in: StockMovement
}

export interface ReceiveStockInput {
  warehouse_id: string
  quantity: number
  batch_id?: string | null
  lot_number?: string
  expiry_date?: string | null
  reason?: string
  occurred_at?: string
}

export interface ReceiveStockResult {
  batch?: Batch | null
  movement: StockMovement
}

export interface MarkDamagedOrLostInput {
  warehouse_id: string
  batch_id?: string | null
  quantity: number
  type: 'damaged' | 'lost'
  reason?: string
  occurred_at?: string
}

export interface ExpiringBatch {
  batch_id: string
  sku_id: string
  sku_code: string
  product_name: string
  warehouse_id: string
  warehouse_name: string
  lot_number: string
  expiry_date: string
  quantity: number
}

export interface ExpiredBatchReview {
  batch_id: string
  sku_id: string
  sku_code: string
  product_name: string
  warehouse_id: string
  warehouse_name: string
  lot_number: string
  expiry_date: string
  quantity: number
}

export interface ExtendBatchExpiryInput {
  new_expiry_date: string
}

export interface DiscardBatchInput {
  warehouse_id: string
  // Omit for a general-ledger (un-lotted) discard.
  batch_id?: string
  quantity: number
  reason?: string
  photo?: File
}

export type DiscardRecordType = 'damaged' | 'lost'

export interface DiscardRecord {
  id: string
  tenant_id: string
  sku_id: string
  warehouse_id: string
  batch_id?: string | null
  movement_id: string
  // Expiry/scanner discards are always 'lost' -- the inventory page's
  // Damaged/Lost button and a booking's write-off replay can be either.
  type: DiscardRecordType
  quantity: number
  reason: string
  photo_url?: string | null
  discarded_by: string
  discarded_at: string
  created_at: string
}

export interface DiscardRecordReport extends DiscardRecord {
  sku_code: string
  product_name: string
  warehouse_name: string
  lot_number: string
}

export interface ListDiscardsParams {
  warehouseId?: string
  skuId?: string
  type?: DiscardRecordType
  from?: string
  to?: string
  limit?: number
  offset?: number
}

export interface ListDiscardsResult {
  discards: DiscardRecordReport[]
  total: number
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
