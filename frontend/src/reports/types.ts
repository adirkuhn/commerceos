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

// winners = high revenue + high margin (push harder); stars = high revenue
// + low margin (optimize pricing); hidden_gems = low revenue + high margin
// (promote); problems = low revenue + low margin (consider discontinuing).
// "High"/"low" are relative to the response's own median_revenue_minor_units
// /median_margin_basis_points, not a fixed threshold.
export type ProductPerformanceQuadrant = 'winners' | 'stars' | 'hidden_gems' | 'problems'

export interface ProductPerformanceMatrixRow extends ProductPerformanceRow {
  quadrant: ProductPerformanceQuadrant
}

export interface ProductPerformanceMatrix {
  currency_code: string
  median_revenue_minor_units: number
  median_margin_basis_points: number
  rows: ProductPerformanceMatrixRow[]
}

export interface CustomerPerformanceRow {
  customer_id: string
  // Empty string if the customer record has no name set.
  customer_name: string
  order_count: number
  revenue_minor_units: number
  average_order_value_minor_units: number
  first_order_at: string
  last_order_at: string
}

export interface CustomerPerformanceReport {
  currency_code: string
  rows: CustomerPerformanceRow[]
}

export interface CategoryPerformanceRow {
  // Null for the "Uncategorized" bucket (adhoc line item, deleted SKU, or
  // product with no category).
  category_id: string | null
  category_name: string
  quantity_sold: number
  revenue_minor_units: number
  order_count: number
}

export interface CategoryPerformanceReport {
  currency_code: string
  rows: CategoryPerformanceRow[]
}

export interface SlowMovingProductRow {
  sku_id: string
  sku_code: string
  product_name: string
  stock_on_hand: number
  // Always 0 -- that's this report's inclusion criterion.
  quantity_sold_in_period: number
}

export interface SlowMovingProductsReport {
  rows: SlowMovingProductRow[]
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
