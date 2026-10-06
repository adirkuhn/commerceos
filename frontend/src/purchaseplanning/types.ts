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
  // Season plan only: forecast until the season starts, and how many
  // times its usual daily rate it sold in last year's season.
  pre_season?: Forecast | null
  seasonal_lift?: number | null
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

export type SeasonEventKey =
  | 'christmas'
  | 'sinterklaas'
  | 'black_friday'
  | 'easter'
  | 'carnival'
  | 'valentines'
  | 'mothers_day'
  | 'fathers_day_br'
  | 'festa_junina'
  | 'dia_dos_namorados'
  | 'fathers_day_nl'
  | 'kings_day'
  | 'halloween'
  | 'dia_das_criancas'
  | 'sint_maarten'
  | 'new_years_eve'

// Calendar days [start, end) -- end is exclusive.
export interface DateWindow {
  start: string
  end: string
}

export interface SeasonEvent {
  key: SeasonEventKey
  year: number
  date: string
  window: DateWindow
}

export interface SeasonPlan {
  as_of: string
  event?: SeasonEventKey | null
  year: number
  window: DateWindow
  last_year_window: DateWindow
  safety_days: number
  suppliers: SupplierRecommendation[]
}

// Either event (+ year) or a custom from/to (to exclusive).
export interface SeasonPlanQuery {
  event?: SeasonEventKey | null
  year?: number | null
  from?: string | null
  to?: string | null
  safety_days: number
  supplier_id?: string | null
  all_products: boolean
}

export interface PlanOrderLine {
  sku_id: string
  // Purchased items (boxes, kg), as the plan shows them.
  quantity: number
}

export interface CreatePlanOrderInput {
  supplier_id: string
  lines: PlanOrderLine[]
  notes?: string
}

export interface CreatedPurchaseOrder {
  purchase_order_id: string
}

export interface APIResponse<T> {
  data: T | null
  error: { code: string; message: string } | null
}
