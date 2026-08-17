import { useState, useEffect, useCallback } from 'react'
import type { DiscardRecordReport, ListDiscardsParams } from '../types'
import * as api from '../api'

interface UseDiscardsState {
  discards: DiscardRecordReport[]
  total: number
  loading: boolean
  error: string | null
}

// The discarded-items report's data source.
export function useDiscards(token: string, params: ListDiscardsParams = {}): UseDiscardsState & { refresh: () => void } {
  const { warehouseId, skuId, from, to, limit, offset } = params
  const [discards, setDiscards] = useState<DiscardRecordReport[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const result = await api.listDiscards(token, { warehouseId, skuId, from, to, limit, offset })
      setDiscards(result.discards)
      setTotal(result.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, warehouseId, skuId, from, to, limit, offset])

  useEffect(() => { load() }, [load])

  return { discards, total, loading, error, refresh: load }
}
