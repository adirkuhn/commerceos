import { useState, useEffect, useCallback } from 'react'
import type { SalesOrder, SalesOrderDetail, CreateSalesOrderInput, SalesOrderSortField, SortDirection } from '../types'
import * as api from '../api'

interface UseSalesOrdersState {
  salesOrders: SalesOrder[]
  total: number
  loading: boolean
  error: string | null
}

interface UseSalesOrdersActions {
  refresh: () => void
  create: (input: CreateSalesOrderInput) => Promise<SalesOrderDetail>
}

export function useSalesOrders(
  token: string,
  params: { page?: number; perPage?: number; sortBy?: SalesOrderSortField; sortDir?: SortDirection } = {},
): UseSalesOrdersState & UseSalesOrdersActions {
  const { page, perPage, sortBy, sortDir } = params
  const [salesOrders, setSalesOrders] = useState<SalesOrder[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const result = await api.listSalesOrders(token, { page, perPage, sortBy, sortDir })
      setSalesOrders(result.salesOrders)
      setTotal(result.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, page, perPage, sortBy, sortDir])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: CreateSalesOrderInput): Promise<SalesOrderDetail> => {
    const result = await api.createSalesOrder(token, input)
    await load()
    return result
  }, [token, load])

  return { salesOrders, total, loading, error, refresh: load, create }
}
