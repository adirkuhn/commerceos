import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import * as pmsApi from '../../pms/api'
import * as omsApi from '../../oms/api'
import * as inventoryApi from '../../inventory/api'
import * as dashboardApi from '../api'
import type { DailyStockFlow, ChannelRevenue, TopProduct } from '../types'

export interface DashboardStats {
  totalProducts: number
  totalOrders: number
  stockIssues: number
  expiringSoon: number
  expired: number
  totalStockUnits: number
  currencyCode: string
  revenueThisMonthMinorUnits: number
  stockFlowLast7Days: DailyStockFlow[]
  salesByChannelThisMonth: ChannelRevenue[]
  topProductsThisMonth: TopProduct[]
}

interface UseDashboardStatsState {
  stats: DashboardStats | null
  loading: boolean
  error: string | null
}

export function useDashboardStats(token: string): UseDashboardStatsState & { refresh: () => void } {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const [products, orders, stockIssueOrders, expiringBatches, expiredBatches, report] = await Promise.all([
        pmsApi.listProducts(token, { perPage: 1 }),
        omsApi.listSalesOrders(token, { perPage: 1 }),
        omsApi.listSalesOrders(token, { perPage: 1, needsStockReview: true }),
        inventoryApi.listExpiringBatches(token, 7),
        inventoryApi.listPendingExpiredBatches(token),
        dashboardApi.getDashboardStats(token),
      ])
      setStats({
        totalProducts: products.total,
        totalOrders: orders.total,
        stockIssues: stockIssueOrders.total,
        expiringSoon: expiringBatches.length,
        expired: expiredBatches.length,
        totalStockUnits: report.total_stock_units,
        currencyCode: report.currency_code,
        revenueThisMonthMinorUnits: report.revenue_this_month_minor_units,
        stockFlowLast7Days: report.stock_flow_last_7_days,
        salesByChannelThisMonth: report.sales_by_channel_this_month,
        topProductsThisMonth: report.top_products_this_month,
      })
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('dashboard:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  return { stats, loading, error, refresh: load }
}
