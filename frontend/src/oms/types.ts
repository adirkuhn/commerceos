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
}

export interface SalesOrderLineItem {
  id: string
  sku_id: string
  name: string
  sku_code: string
  quantity: number
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
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
