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

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
