import { useState, useEffect, useCallback } from 'react'
import type { SalesOrderDetail } from '../types'
import * as api from '../api'

interface UseSalesOrderDetailState {
  detail: SalesOrderDetail | null
  loading: boolean
  error: string | null
}

interface UseSalesOrderDetailActions {
  refresh: () => void
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
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, salesOrderId])

  useEffect(() => { load() }, [load])

  return { detail, loading, error, refresh: load }
}
