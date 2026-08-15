import { useState, useEffect, useCallback } from 'react'
import type { ExpiredBatchReview } from '../types'
import * as api from '../api'

interface UseExpiredBatchReviewState {
  batches: ExpiredBatchReview[]
  loading: boolean
  error: string | null
}

interface UseExpiredBatchReviewActions {
  refresh: () => void
  writeOff: (batchId: string) => Promise<void>
  extend: (batchId: string, newExpiryDate: string) => Promise<void>
  dismiss: (batchId: string) => Promise<void>
}

// Every action reloads the queue afterwards rather than patching state
// locally -- simplest correct behavior for a queue where the acted-on row
// just needs to disappear.
export function useExpiredBatchReview(token: string): UseExpiredBatchReviewState & UseExpiredBatchReviewActions {
  const [batches, setBatches] = useState<ExpiredBatchReview[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listPendingExpiredBatches(token)
      setBatches(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const writeOff = useCallback(async (batchId: string) => {
    await api.writeOffExpiredBatch(token, batchId)
    await load()
  }, [token, load])

  const extend = useCallback(async (batchId: string, newExpiryDate: string) => {
    await api.extendBatchExpiry(token, batchId, { new_expiry_date: newExpiryDate })
    await load()
  }, [token, load])

  const dismiss = useCallback(async (batchId: string) => {
    await api.dismissExpiredBatch(token, batchId)
    await load()
  }, [token, load])

  return { batches, loading, error, refresh: load, writeOff, extend, dismiss }
}
