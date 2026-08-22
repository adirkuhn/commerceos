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

export type SalesOrderStatus = 'pending' | 'processing' | 'on-hold' | 'completed' | 'cancelled' | 'refunded' | 'failed'

export type SalesOrderSortField = 'occurred_at' | 'customer' | 'source' | 'status' | 'total'
export type SortDirection = 'asc' | 'desc'

export interface SalesOrder {
  id: string
  tenant_id: string
  source: string
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
  total_tax_minor_units: number
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

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
