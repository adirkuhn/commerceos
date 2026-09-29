import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import type { OrderQueue, QueueEntry } from '../types'
import * as api from '../api'

// Locks change as people start/finish orders -- re-poll so "who has it and
// for how long" stays current without a manual refresh.
export const PACKING_QUEUE_POLL_MS = 30_000
const QUEUE_PAGE_SIZE = 100

interface UsePackingQueue {
  entries: QueueEntry[]
  loading: boolean
  error: string
  refresh: () => Promise<void>
}

type QueueFetcher = (token: string, params: { perPage: number }) => Promise<OrderQueue>

export function usePackingQueue(token: string): UsePackingQueue {
  return useOrderQueue(token, api.listPackingQueue)
}

// The picking screens' queue -- same shape, with in_packing set for orders
// packing has.
export function usePickingQueue(token: string): UsePackingQueue {
  return useOrderQueue(token, api.listPickingQueue)
}

function useOrderQueue(token: string, fetchQueue: QueueFetcher): UsePackingQueue {
  const [entries, setEntries] = useState<QueueEntry[]>([])
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState('')

  const refresh = useCallback(async () => {
    if (!token) return
    try {
      const queue = await fetchQueue(token, { perPage: QUEUE_PAGE_SIZE })
      setEntries(queue.entries)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : i18next.t('orders:errors.loadOrders'))
    } finally {
      setLoading(false)
    }
  }, [token, fetchQueue])

  useEffect(() => {
    refresh()
    const interval = setInterval(refresh, PACKING_QUEUE_POLL_MS)
    return () => clearInterval(interval)
  }, [refresh])

  return { entries, loading, error, refresh }
}
