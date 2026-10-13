// The close-to-expiry API (backend api/stockexpiry/openapi.yaml).

export interface CloseToExpiryWindow {
  window_days: number
  // Smallest days-left value in the window (with 90 and 30 in use, the
  // 90-day window holds 31-90 days left).
  min_days_left: number
  from: string
  to: string
  open_count: number
  acknowledged_count: number
}

export interface ClearancePromotionRef {
  id: string
  name: string
}

export interface CloseToExpiryBatch {
  batch_id: string
  lot_number: string
  expiry_date: string
  days_left: number
  quantity: number
  warehouse_id: string
  warehouse_name: string
  sku_id: string
  sku_code: string
  product_id: string
  product_name: string
  image_url: string
  price_minor_units: number
  // The SKU's sale price when a sale runs today.
  sale_price_minor_units?: number | null
  currency_code: string
  acknowledged_at?: string | null
  // The promotion still selling this batch off, if any.
  clearance_promotion?: ClearancePromotionRef | null
  // What one unit of this lot cost: its own landed cost ('batch'), else
  // the SKU's average ('sku'); null when unknown. Per gram for a weight
  // pool. Optional so older fixtures stay valid -- a real response always
  // includes these.
  unit_cost_minor_units?: number | null
  unit_cost_source?: UnitCostSource | null
  // The SKU's tax class; null = the shop's default class.
  tax_class_id?: string | null
  // A kilo SKU: stock in grams, never sold itself (its packages are).
  weight_pool?: boolean
}

export type UnitCostSource = 'batch' | 'sku'

export interface CloseToExpiryList {
  today: string
  window: CloseToExpiryWindow
  // Every window in use, largest first.
  windows: CloseToExpiryWindow[]
  items: CloseToExpiryBatch[]
  page: number
  per_page: number
  total: number
}

export interface CloseToExpiryQuery {
  window?: number
  acknowledged?: boolean
  page?: number
  perPage?: number
}
