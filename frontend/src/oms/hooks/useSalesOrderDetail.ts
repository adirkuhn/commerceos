import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { SalesOrderDetail, SalesOrder, UpdateSalesOrderInput } from '../types'
import * as api from '../api'

interface UseSalesOrderDetailState {
  detail: SalesOrderDetail | null
  loading: boolean
  error: string | null
}

interface UseSalesOrderDetailActions {
  refresh: () => void
  update: (input: UpdateSalesOrderInput) => Promise<SalesOrder>
  remove: () => Promise<void>
}

export function useSalesOrderDetail(token: string, salesOrderId: string | null): UseSalesOrderDetailState & UseSalesOrderDetailActions {
  const [detail, setDetail] = useState<SalesOrderDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !salesOrderId) {
      setDetail(null)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const data = await api.getSalesOrder(token, salesOrderId)
      setDetail(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('orders:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token, salesOrderId])

  useEffect(() => { load() }, [load])

  const update = useCallback(async (input: UpdateSalesOrderInput): Promise<SalesOrder> => {
    if (!salesOrderId) throw new Error('no sales order selected')
    const order = await api.updateSalesOrder(token, salesOrderId, input)
    await load()
    return order
  }, [token, salesOrderId, load])

  const remove = useCallback(async (): Promise<void> => {
    if (!salesOrderId) throw new Error('no sales order selected')
    await api.deleteSalesOrder(token, salesOrderId)
  }, [token, salesOrderId])

  return { detail, loading, error, refresh: load, update, remove }
}
