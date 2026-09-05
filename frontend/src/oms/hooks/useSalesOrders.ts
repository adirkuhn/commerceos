import { useState, useEffect, useCallback } from 'react'
import type { SalesOrder, SalesOrderDetail, CreateSalesOrderInput, SalesOrderSortField, SortDirection, SalesOrderSource, SalesOrderStatus } from '../types'
import * as api from '../api'
import type { BulkUpdateSalesOrdersInput, BulkUpdateSalesOrdersResult } from '../api'

interface UseSalesOrdersState {
  salesOrders: SalesOrder[]
  total: number
  loading: boolean
  error: string | null
}

interface UseSalesOrdersActions {
  refresh: () => void
  create: (input: CreateSalesOrderInput) => Promise<SalesOrderDetail>
  bulkUpdate: (input: BulkUpdateSalesOrdersInput) => Promise<BulkUpdateSalesOrdersResult>
}

export function useSalesOrders(
  token: string,
  params: {
    page?: number
    perPage?: number
    sortBy?: SalesOrderSortField
    sortDir?: SortDirection
    // needsStockReview/needsSkuReview: see api.ListSalesOrdersParams --
    // used by the "Stock Issues"/"Unmatched SKUs" pages to narrow the list.
    needsStockReview?: boolean
    needsSkuReview?: boolean
    // fulfilled: false powers the picking scanner app's "orders to pick"
    // search; also the Sales Orders page's own substatus filter (nested
    // under status).
    fulfilled?: boolean
    // status/source: the Sales Orders page's own filters.
    status?: SalesOrderStatus[]
    source?: SalesOrderSource[]
    // customerId: narrows to one customer's orders -- the Customer detail
    // page's order history and lifetime-spend figures.
    customerId?: string
  } = {},
): UseSalesOrdersState & UseSalesOrdersActions {
  const { page, perPage, sortBy, sortDir, needsStockReview, needsSkuReview, fulfilled, status, source, customerId } = params
  const [salesOrders, setSalesOrders] = useState<SalesOrder[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const result = await api.listSalesOrders(token, { page, perPage, sortBy, sortDir, needsStockReview, needsSkuReview, fulfilled, status, source, customerId })
      setSalesOrders(result.salesOrders)
      setTotal(result.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, page, perPage, sortBy, sortDir, needsStockReview, needsSkuReview, fulfilled, status, source, customerId])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: CreateSalesOrderInput): Promise<SalesOrderDetail> => {
    const result = await api.createSalesOrder(token, input)
    await load()
    return result
  }, [token, load])

  const bulkUpdate = useCallback(async (input: BulkUpdateSalesOrdersInput): Promise<BulkUpdateSalesOrdersResult> => {
    const result = await api.bulkUpdateSalesOrders(token, input)
    await load()
    return result
  }, [token, load])

  return { salesOrders, total, loading, error, refresh: load, create, bulkUpdate }
}
