export interface SalesSummary {
  revenue_minor_units: number
  order_count: number
  units_sold: number
  average_order_value_minor_units: number
}

export interface SalesSummaryReport {
  currency_code: string
  current: SalesSummary
  previous: SalesSummary
  // Null when the previous period's corresponding figure was zero -- no
  // meaningful percent change off a zero baseline.
  revenue_change_percent: number | null
  order_count_change_percent: number | null
  units_sold_change_percent: number | null
  aov_change_percent: number | null
}

export interface ProductPerformanceRow {
  sku_id: string | null
  sku_code: string
  product_name: string
  quantity_sold: number
  revenue_minor_units: number
  order_count: number
  first_sale_at: string
  last_sale_at: string
  // Null together when there's no cost basis to join (adhoc line item, SKU
  // since deleted, or cost price never computed).
  cost_price_minor_units: number | null
  profit_minor_units: number | null
  margin_basis_points: number | null
}

export interface ProductPerformanceReport {
  currency_code: string
  rows: ProductPerformanceRow[]
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
