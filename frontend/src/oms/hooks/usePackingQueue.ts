import { useCallback, useEffect, useState } from 'react'
import type { PackingQueueEntry } from '../types'
import * as api from '../api'

// Locks change as people start/finish orders -- re-poll so "who has it and
// for how long" stays current without a manual refresh.
export const PACKING_QUEUE_POLL_MS = 30_000
const QUEUE_PAGE_SIZE = 100

interface UsePackingQueue {
  entries: PackingQueueEntry[]
  loading: boolean
  error: string
  refresh: () => Promise<void>
}

export function usePackingQueue(token: string): UsePackingQueue {
  const [entries, setEntries] = useState<PackingQueueEntry[]>([])
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState('')

  const refresh = useCallback(async () => {
    if (!token) return
    try {
      const queue = await api.listPackingQueue(token, { perPage: QUEUE_PAGE_SIZE })
      setEntries(queue.entries)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load orders')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => {
    refresh()
    const interval = setInterval(refresh, PACKING_QUEUE_POLL_MS)
    return () => clearInterval(interval)
  }, [refresh])

  return { entries, loading, error, refresh }
}
