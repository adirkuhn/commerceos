export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}

export interface DailyStockFlow {
  date: string
  in: number
  out: number
}

export interface ChannelRevenue {
  source: string
  total_minor_units: number
}

export interface TopProduct {
  sku_id: string | null
  sku_code: string
  product_name: string
  quantity_sold: number
  revenue_minor_units: number
}

export interface DashboardStatsReport {
  total_stock_units: number
  currency_code: string
  revenue_this_month_minor_units: number
  stock_flow_last_7_days: DailyStockFlow[]
  sales_by_channel_this_month: ChannelRevenue[]
  top_products_this_month: TopProduct[]
}

// GET /dashboard/today -- the Today work-queue counts (backend
// api/dashboard/openapi.yaml TodayQueue).
export interface TodayQueue {
  to_pick: number
  to_pack: number
  stock_issues: number
  unmatched_skus: number
  expiring_batches: number
  expired_batches: number
}
