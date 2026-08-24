import { useState, useEffect, useCallback } from 'react'
import * as reportsApi from '../api'
import * as dashboardApi from '../../dashboard/api'
import type { SalesSummaryReport, ProductPerformanceReport, ProductPerformanceMatrix, CustomerPerformanceReport, CategoryPerformanceReport, SlowMovingProductsReport, ProfitabilityReport, InventorySalesReport, BusinessInsightsReport, SalesTrendsReport, SalesTrendGranularity, ProductLifecycleReport, CustomerCohortsReport, CustomerRFMReport, FrequentlyBoughtTogetherReport, StockoutRiskReport } from '../types'
import type { ChannelRevenue } from '../../dashboard/types'

export type ReportsPeriod = '7D' | '30D' | '90D' | '12M'

const PERIOD_DAYS: Record<ReportsPeriod, number> = {
  '7D': 7,
  '30D': 30,
  '90D': 90,
  '12M': 365,
}

// dateRange returns [from, to) for period, ending at "now" -- the sales-
// summary/product-performance endpoints compare it against the immediately
// preceding period of equal length themselves, so the caller only ever
// needs this one range.
export function dateRange(period: ReportsPeriod, now: Date): { from: string; to: string } {
  const to = now
  const from = new Date(now.getTime() - PERIOD_DAYS[period] * 24 * 60 * 60 * 1000)
  return { from: from.toISOString(), to: to.toISOString() }
}

interface UseReportsDataState {
  salesSummary: SalesSummaryReport | null
  productPerformance: ProductPerformanceReport | null
  productPerformanceMatrix: ProductPerformanceMatrix | null
  customerPerformance: CustomerPerformanceReport | null
  categoryPerformance: CategoryPerformanceReport | null
  slowMovingProducts: SlowMovingProductsReport | null
  profitability: ProfitabilityReport | null
  inventorySales: InventorySalesReport | null
  businessInsights: BusinessInsightsReport | null
  salesTrends: SalesTrendsReport | null
  productLifecycle: ProductLifecycleReport | null
  customerCohorts: CustomerCohortsReport | null
  customerRFM: CustomerRFMReport | null
  frequentlyBoughtTogether: FrequentlyBoughtTogetherReport | null
  stockoutRisk: StockoutRiskReport | null
  // Real revenue-by-channel data, but always "this month" (the underlying
  // /dashboard/stats endpoint has no date-range parameter) -- not scoped
  // to the selected period. Labeled as such in the UI.
  channelRevenueThisMonth: ChannelRevenue[]
  loading: boolean
  error: string | null
}

export function useReportsData(
  token: string,
  period: ReportsPeriod,
  trendGranularity: SalesTrendGranularity = 'day',
): UseReportsDataState & { refresh: () => void } {
  const [salesSummary, setSalesSummary] = useState<SalesSummaryReport | null>(null)
  const [productPerformance, setProductPerformance] = useState<ProductPerformanceReport | null>(null)
  const [productPerformanceMatrix, setProductPerformanceMatrix] = useState<ProductPerformanceMatrix | null>(null)
  const [customerPerformance, setCustomerPerformance] = useState<CustomerPerformanceReport | null>(null)
  const [categoryPerformance, setCategoryPerformance] = useState<CategoryPerformanceReport | null>(null)
  const [slowMovingProducts, setSlowMovingProducts] = useState<SlowMovingProductsReport | null>(null)
  const [profitability, setProfitability] = useState<ProfitabilityReport | null>(null)
  const [inventorySales, setInventorySales] = useState<InventorySalesReport | null>(null)
  const [businessInsights, setBusinessInsights] = useState<BusinessInsightsReport | null>(null)
  const [salesTrends, setSalesTrends] = useState<SalesTrendsReport | null>(null)
  const [productLifecycle, setProductLifecycle] = useState<ProductLifecycleReport | null>(null)
  const [customerCohorts, setCustomerCohorts] = useState<CustomerCohortsReport | null>(null)
  const [customerRFM, setCustomerRFM] = useState<CustomerRFMReport | null>(null)
  const [frequentlyBoughtTogether, setFrequentlyBoughtTogether] = useState<FrequentlyBoughtTogetherReport | null>(null)
  const [stockoutRisk, setStockoutRisk] = useState<StockoutRiskReport | null>(null)
  const [channelRevenueThisMonth, setChannelRevenueThisMonth] = useState<ChannelRevenue[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const { from, to } = dateRange(period, new Date())
      const [summary, performance, matrix, customers, categories, slowMoving, profitabilityReport, inventorySalesReport, insightsReport, trendsReport, lifecycleReport, cohortsReport, rfmReport, frequentlyBoughtTogetherReport, stockoutRiskReport, dashboardStats] = await Promise.all([
        reportsApi.getSalesSummaryReport(token, from, to),
        reportsApi.getProductPerformanceReport(token, from, to),
        reportsApi.getProductPerformanceMatrix(token, from, to),
        reportsApi.getCustomerPerformanceReport(token, from, to),
        reportsApi.getCategoryPerformanceReport(token, from, to),
        reportsApi.getSlowMovingProductsReport(token, from, to),
        reportsApi.getProfitabilityReport(token, from, to),
        reportsApi.getInventorySalesReport(token, from, to),
        reportsApi.getBusinessInsights(token, from, to),
        reportsApi.getSalesTrendsReport(token, from, to, trendGranularity),
        reportsApi.getProductLifecycleReport(token, from, to),
        reportsApi.getCustomerCohortsReport(token, from, to),
        reportsApi.getCustomerRFMReport(token, from, to),
        reportsApi.getFrequentlyBoughtTogetherReport(token, from, to),
        reportsApi.getStockoutRiskReport(token, from, to),
        dashboardApi.getDashboardStats(token),
      ])
      setSalesSummary(summary)
      setProductPerformance(performance)
      setProductPerformanceMatrix(matrix)
      setCustomerPerformance(customers)
      setCategoryPerformance(categories)
      setSlowMovingProducts(slowMoving)
      setProfitability(profitabilityReport)
      setInventorySales(inventorySalesReport)
      setBusinessInsights(insightsReport)
      setSalesTrends(trendsReport)
      setProductLifecycle(lifecycleReport)
      setCustomerCohorts(cohortsReport)
      setCustomerRFM(rfmReport)
      setFrequentlyBoughtTogether(frequentlyBoughtTogetherReport)
      setStockoutRisk(stockoutRiskReport)
      setChannelRevenueThisMonth(dashboardStats.sales_by_channel_this_month)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, period, trendGranularity])

  useEffect(() => { load() }, [load])

  return { salesSummary, productPerformance, productPerformanceMatrix, customerPerformance, categoryPerformance, slowMovingProducts, profitability, inventorySales, businessInsights, salesTrends, productLifecycle, customerCohorts, customerRFM, frequentlyBoughtTogether, stockoutRisk, channelRevenueThisMonth, loading, error, refresh: load }
}
