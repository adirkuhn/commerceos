import { useState, useEffect, useCallback } from 'react'
import type { SalesOrder } from '../types'
import * as api from '../api'

interface UseSalesOrdersState {
  salesOrders: SalesOrder[]
  total: number
  loading: boolean
  error: string | null
}

interface UseSalesOrdersActions {
  refresh: () => void
}

export function useSalesOrders(token: string, params: { page?: number; perPage?: number } = {}): UseSalesOrdersState & UseSalesOrdersActions {
  const { page, perPage } = params
  const [salesOrders, setSalesOrders] = useState<SalesOrder[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const result = await api.listSalesOrders(token, { page, perPage })
      setSalesOrders(result.salesOrders)
      setTotal(result.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, page, perPage])

  useEffect(() => { load() }, [load])

  return { salesOrders, total, loading, error, refresh: load }
}
