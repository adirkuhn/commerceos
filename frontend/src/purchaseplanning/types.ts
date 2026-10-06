// Mirrors backend/api/purchaseplanning/openapi.yaml.

export type ForecastMethod = 'seasonal' | 'recent_velocity' | 'no_demand'

export interface Forecast {
  units: number
  method: ForecastMethod
  growth_factor: number
  days: number
  recent_units?: number
  last_year_units?: number
}

export interface OnOrder {
  sent: number
  draft: number
}

// Quantities are stock units -- grams for a weight pool -- except
// purchase_quantity, which is in purchased items (boxes, kg).
export interface RecommendationLine {
  sku_id: string
  sku_code: string
  product_name: string
  variant_name: string
  is_weight_pool: boolean
  on_hand: number
  on_order: OnOrder
  forecast: Forecast
  daily_demand: number
  safety_stock: number
  needed_units: number
  purchase_quantity: number
  units_per_purchased_item: number
  order_units: number
  unit_cost_minor_units?: number | null
  estimated_cost_minor_units?: number | null
  currency_code: string
  stockout_date?: string | null
  order_by_date?: string | null
  last_ordered_at?: string | null
}

export interface SupplierRecommendation {
  // Null for SKUs never bought on a purchase order.
  supplier_id?: string | null
  supplier_name: string
  lead_time_days: number
  lead_time_assumed: boolean
  lines: RecommendationLine[]
  estimated_cost_minor_units?: number | null
  currency_code: string
  earliest_order_by_date?: string | null
}

export interface ReorderRecommendations {
  as_of: string
  coverage_days: number
  safety_days: number
  suppliers: SupplierRecommendation[]
}

export interface ReorderRecommendationsQuery {
  coverage_days: number
  safety_days: number
  supplier_id?: string | null
}

export interface APIResponse<T> {
  data: T | null
  error: { code: string; message: string } | null
}
