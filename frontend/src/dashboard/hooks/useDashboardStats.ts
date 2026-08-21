import { useState, useEffect, useCallback } from 'react'
import * as pmsApi from '../../pms/api'
import * as omsApi from '../../oms/api'
import * as inventoryApi from '../../inventory/api'

export interface DashboardStats {
  totalProducts: number
  totalOrders: number
  stockIssues: number
  expiringSoon: number
  expired: number
}

interface UseDashboardStatsState {
  stats: DashboardStats | null
  loading: boolean
  error: string | null
}

// Each figure here comes straight from an existing list endpoint's total/count
// -- there is no tenant-wide reporting/aggregate backend yet, so anything
// that would need one (revenue, stock-flow, channel/product breakdowns)
// is deliberately left off this hook rather than approximated client-side.
export function useDashboardStats(token: string): UseDashboardStatsState & { refresh: () => void } {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const [products, orders, stockIssueOrders, expiringBatches, expiredBatches] = await Promise.all([
        pmsApi.listProducts(token, { perPage: 1 }),
        omsApi.listSalesOrders(token, { perPage: 1 }),
        omsApi.listSalesOrders(token, { perPage: 1, needsStockReview: true }),
        inventoryApi.listExpiringBatches(token, 7),
        inventoryApi.listPendingExpiredBatches(token),
      ])
      setStats({
        totalProducts: products.total,
        totalOrders: orders.total,
        stockIssues: stockIssueOrders.total,
        expiringSoon: expiringBatches.length,
        expired: expiredBatches.length,
      })
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  return { stats, loading, error, refresh: load }
}
