import { useState, useEffect, useCallback } from 'react'
import type { ExpiringBatch } from '../types'
import * as api from '../api'

interface UseExpiringBatchesState {
  batches: ExpiringBatch[]
  loading: boolean
  error: string | null
}

export function useExpiringBatches(token: string, withinDays = 30): UseExpiringBatchesState & { refresh: () => void } {
  const [batches, setBatches] = useState<ExpiringBatch[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listExpiringBatches(token, withinDays)
      setBatches(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, withinDays])

  useEffect(() => { load() }, [load])

  return { batches, loading, error, refresh: load }
}
