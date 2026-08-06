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

export type MovementType = 'receipt' | 'sale' | 'adjustment' | 'transfer_out' | 'transfer_in' | 'damaged' | 'lost'

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

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
