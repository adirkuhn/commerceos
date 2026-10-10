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

export interface ProfitabilitySummary {
  revenue_minor_units: number
  // Subset of revenue_minor_units whose SKU has a known cost price --
  // what profit_minor_units/margin_basis_points actually cover.
  cost_known_revenue_minor_units: number
  cost_minor_units: number
  profit_minor_units: number
  // 0 when cost_known_revenue_minor_units is 0. 1 bp = 0.01%.
  margin_basis_points: number
}

export interface ProfitabilityReport {
  currency_code: string
  summary: ProfitabilitySummary
}

export interface ProfitabilityPoint {
  period_start: string
  revenue_minor_units: number
  cost_known_revenue_minor_units: number
  cost_minor_units: number
  profit_minor_units: number
  // 0 when cost_known_revenue_minor_units is 0 for this bucket. 1 bp = 0.01%.
  margin_basis_points: number
}

export interface ProfitabilityTrendReport {
  currency_code: string
  granularity: SalesTrendGranularity
  points: ProfitabilityPoint[]
}

export interface ChannelRevenue {
  // Sales order source -- pos, manual, or woocommerce.
  source: string
  total_minor_units: number
}

export interface ChannelRevenuePoint {
  period_start: string
  source: string
  revenue_minor_units: number
}

export interface RevenueByChannelReport {
  currency_code: string
  channels: ChannelRevenue[]
  granularity: SalesTrendGranularity
  points: ChannelRevenuePoint[]
}

export interface InventorySalesRow {
  sku_id: string
  sku_code: string
  product_name: string
  stock_on_hand: number
  quantity_sold_in_period: number
  // Units sold as a fraction of current stock on hand. 1 bp = 0.01%, e.g.
  // 2500 = 25.00%; can exceed 10000 if restocked mid-period. Null when
  // stock_on_hand is 0.
  turnover_basis_points: number | null
}

export interface InventorySalesReport {
  rows: InventorySalesRow[]
}

export type InsightSeverity = 'info' | 'warning'

export interface Insight {
  // Stable machine-readable id for the rule that fired, e.g. "revenue_declining".
  code: string
  severity: InsightSeverity
  // The rule's numbers ({count} or {percent}) -- see insightText.
  params?: Record<string, unknown>
  // English fallback.
  message: string
}

export interface BusinessInsightsReport {
  insights: Insight[]
}

export type SalesTrendGranularity = 'day' | 'week' | 'month'

export interface SalesTrendPoint {
  period_start: string
  revenue_minor_units: number
  order_count: number
  units_sold: number
}

export interface WeekdaySales {
  // 0 = Sunday .. 6 = Saturday.
  weekday: number
  revenue_minor_units: number
  order_count: number
}

export interface SalesTrendsReport {
  currency_code: string
  granularity: SalesTrendGranularity
  points: SalesTrendPoint[]
  by_weekday: WeekdaySales[]
}

export type ProductLifecycleStage = 'new' | 'growing' | 'stable' | 'declining' | 'dormant'

export interface ProductLifecycleRow {
  sku_id: string | null
  sku_code: string
  product_name: string
  current_quantity_sold: number
  previous_quantity_sold: number
  current_revenue_minor_units: number
  previous_revenue_minor_units: number
  stage: ProductLifecycleStage
}

export interface ProductLifecycleReport {
  currency_code: string
  rows: ProductLifecycleRow[]
}

export interface CohortRetentionCell {
  // 0 = the acquisition month itself.
  months_since_cohort: number
  retained_customers: number
  revenue_minor_units: number
  // retained_customers / the cohort's size. 1 bp = 0.01%. 0 when the cohort is empty.
  retention_rate_basis_points: number
}

export interface CustomerCohortRow {
  cohort_month: string
  cohort_size: number
  retention: CohortRetentionCell[]
}

export interface CustomerCohortsReport {
  currency_code: string
  cohorts: CustomerCohortRow[]
}

export interface PromotionPerformancePeriod {
  from: string
  to: string
  revenue_minor_units: number
  units_sold: number
  order_count: number
}

export interface PromotionPerformanceReport {
  promotion_id: string
  promotion_name: string
  currency_code: string
  before: PromotionPerformancePeriod
  during: PromotionPerformancePeriod
  // Null when the before period's revenue was zero (no meaningful percent change off a zero baseline).
  revenue_change_percent: number | null
  units_sold_change_percent: number | null
}

export type RFMSegment = 'champions' | 'loyal_customers' | 'potential_loyalists' | 'at_risk' | 'lost' | 'needs_attention'

export interface CustomerRFMRow {
  customer_id: string
  // Empty string if the customer record has no name set.
  customer_name: string
  order_count: number
  revenue_minor_units: number
  average_order_value_minor_units: number
  first_order_at: string
  last_order_at: string
  // 1-5 quintile, 5 = best, relative to the other customers in this result set.
  recency_score: number
  frequency_score: number
  monetary_score: number
  segment: RFMSegment
}

export interface CustomerRFMReport {
  currency_code: string
  rows: CustomerRFMRow[]
}

export interface FrequentlyBoughtTogetherItem {
  sku_id: string
  // Empty if the SKU has since been deleted.
  sku_code: string
  product_name: string
}

export interface FrequentlyBoughtTogetherRow {
  items: FrequentlyBoughtTogetherItem[]
  support: number
  support_percent: number
}

export interface FrequentlyBoughtTogetherReport {
  rows: FrequentlyBoughtTogetherRow[]
  total_orders: number
}

export interface StockoutRiskRow {
  sku_id: string
  sku_code: string
  product_name: string
  stock_on_hand: number
  quantity_sold_in_period: number
  daily_velocity: number
  // Null when daily_velocity is 0 -- no recent demand to project from.
  days_of_stock_remaining: number | null
  at_risk: boolean
  suggested_reorder_quantity: number
}

export interface StockoutRiskReport {
  rows: StockoutRiskRow[]
}

export interface ProductRecommendationRow {
  sku_id: string
  // Empty if the SKU has since been deleted.
  sku_code: string
  product_name: string
  support: number
}

export interface ProductRecommendationsReport {
  sku_id: string
  rows: ProductRecommendationRow[]
  total_orders: number
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
