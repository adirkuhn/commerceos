// Mirrors backend/api/oms/openapi.yaml
export interface Address {
  first_name?: string
  last_name?: string
  company?: string
  address_1?: string
  address_2?: string
  city?: string
  state?: string
  postcode?: string
  country?: string
  email?: string
  phone?: string
}

export type SalesOrderStatus = 'pending' | 'processing' | 'on-hold' | 'shipped' | 'completed' | 'cancelled' | 'refunded' | 'failed'
export type SalesOrderSource = 'pos' | 'manual' | 'woocommerce' | 'bol'

export type SalesOrderSortField = 'occurred_at' | 'customer' | 'source' | 'status' | 'total'
export type SortDirection = 'asc' | 'desc'

export interface SalesOrder {
  id: string
  tenant_id: string
  source: SalesOrderSource
  external_order_id: string
  number: string
  order_key: string
  status: SalesOrderStatus
  warehouse_id?: string | null
  customer_id?: string | null
  billing?: Address
  shipping?: Address
  customer_note?: string
  currency_code: string
  discount_total_minor_units: number
  shipping_total_minor_units: number
  tax_total_minor_units: number
  total_minor_units: number
  payment_method?: string
  payment_method_title?: string
  shipping_method?: string
  shipping_method_title?: string
  transaction_id?: string
  date_paid?: string | null
  date_completed?: string | null
  occurred_at: string
  created_at: string
  // needs_stock_review: at least one linked (sku_id set) line item's stock
  // decrement is pending -- it hit insufficient stock and is waiting on the
  // "Stock Issues" page's retry action. needs_sku_review: at least one line
  // item was never matched to a catalog SKU -- waiting on the
  // "Unmatched SKUs" page's link action. An order can carry both flags.
  needs_stock_review?: boolean
  needs_sku_review?: boolean
  // Binary secondary status, independent of status -- true once every item
  // has physically shipped/been picked. false powers the picking scanner
  // app's "orders to pick" search.
  fulfilled?: boolean
  // Fulfilled's packing-phase counterpart -- true once a PackingSession has
  // completed for this order (which also sets fulfilled: picking is
  // optional, an order may be packed without ever being picked).
  packed?: boolean
}

export type StockStatus = 'decremented' | 'pending'

export interface SalesOrderLineItem {
  id: string
  // Null when this line came from an imported order (e.g. WooCommerce)
  // whose SKU string never matched any CommerceOS SKU -- name/sku_code/
  // price/tax below are still populated from whatever the source
  // provided, so the line reads correctly even unlinked.
  sku_id?: string | null
  name: string
  sku_code: string
  barcode: string
  quantity: number
  // pending: this line's stock decrement is owed (its order's status
  // reduces stock) but hasn't happened -- either sku_id is still null, or
  // the decrement was attempted and hit insufficient stock.
  stock_status?: StockStatus
  price_minor_units: number
  subtotal_minor_units: number
  subtotal_tax_minor_units: number
  total_minor_units: number
  // How much of quantity has been refunded -- the rest is still owed.
  // Optional: older fixtures omit it; missing means 0.
  refunded_quantity?: number
  total_tax_minor_units: number
  // What WooCommerce reported as the expiry date sold, snapshotted at
  // import time from the commerceos-woo plugin's order-item meta -- null
  // for every non-Woo line, or a Woo line whose product had no expiry meta
  // set at checkout. This is Woo's own belief, not necessarily the batch
  // CommerceOS's own FEFO allocation actually decremented.
  sold_expiry_date?: string | null
}

export interface SalesOrderDetail {
  sales_order: SalesOrder
  line_items: SalesOrderLineItem[]
}

export interface CreateSalesOrderLineItemInput {
  sku_id: string
  name?: string
  sku_code?: string
  quantity: number
  price_minor_units: number
  subtotal_minor_units?: number
  subtotal_tax_minor_units?: number
  total_minor_units?: number
  total_tax_minor_units?: number
}

export interface CreateSalesOrderInput {
  number?: string
  order_key?: string
  status?: SalesOrderStatus
  warehouse_id: string
  customer_id?: string | null
  billing?: Address
  shipping?: Address
  customer_note?: string
  currency_code: string
  discount_total_minor_units?: number
  shipping_total_minor_units?: number
  tax_total_minor_units?: number
  total_minor_units?: number
  payment_method?: string
  payment_method_title?: string
  shipping_method?: string
  shipping_method_title?: string
  transaction_id?: string
  date_paid?: string | null
  date_completed?: string | null
  occurred_at: string
  line_items: CreateSalesOrderLineItemInput[]
}

// Header-only edit -- line items are immutable once a sales order is
// created, see backend's UpdateSalesOrderUseCase doc.
export interface UpdateSalesOrderInput {
  number?: string
  order_key?: string
  status?: SalesOrderStatus
  customer_id?: string
  billing?: Address
  shipping?: Address
  customer_note?: string
  payment_method?: string
  payment_method_title?: string
  shipping_method?: string
  shipping_method_title?: string
  transaction_id?: string
  date_paid?: string
  date_completed?: string
  // Manual override -- set explicitly to flip fulfillment status by hand.
  // Normally set by completePicking instead.
  fulfilled?: boolean
}

export interface LinkLineItemSKUInput {
  sku_id: string
}

export interface RetryStockDecrementResult {
  sales_order: SalesOrder
  // resolved: line items whose decrement succeeded this attempt.
  resolved: number
  // still_pending: line items still insufficient after retrying.
  still_pending: number
}

export interface ForceAcceptStockResult {
  sales_order: SalesOrder
  // resolved: SKU-linked pending line items reconciled and decremented.
  resolved: number
}

// SalesOrderActivity is one entry in an order's structured activity log --
// separate from the single freeform customer_note field, used for
// system-recorded events (MyParcel order export, shipment creation, label
// print, and their failures), each tied to the acting user.
export interface SalesOrderActivity {
  id: string
  sales_order_id: string
  user_id: string
  activity_type: string
  summary: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}

// PickList mirrors api/oms's PickList -- a read-only "what to fetch off the
// shelves" view across several orders, summed per SKU. sku_id null means
// the order line was never linked to a catalog SKU (grouped by name).
export interface PickListOrderLine {
  sales_order_id: string
  order_number: string
  line_item_id: string
  quantity: number
}

export interface PickListItem {
  sku_id: string | null
  sku_code: string
  barcode: string
  name: string
  total_quantity: number
  orders: PickListOrderLine[]
}

export interface PickList {
  items: PickListItem[]
}

// PackingQueue mirrors api/oms's PackingQueue -- every order still to be
// packed, each with whoever is actively working it (lock null when free).
export type PackingQueueLockActivity = 'picking' | 'packing'

export interface PackingQueueLock {
  activity: PackingQueueLockActivity
  user_id: string
  since: string
}

export interface PackingQueueEntry {
  sales_order: SalesOrder
  lock: PackingQueueLock | null
}

export interface PackingQueue {
  entries: PackingQueueEntry[]
  total: number
}
