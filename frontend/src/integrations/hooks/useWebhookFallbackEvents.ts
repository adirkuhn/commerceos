import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { WebhookFallbackEvent } from '../types'
import * as api from '../api'

const PER_PAGE = 20

interface UseWebhookFallbackEventsState {
  events: WebhookFallbackEvent[]
  total: number
  page: number
  perPage: number
  loading: boolean
  error: string | null
}

interface UseWebhookFallbackEventsActions {
  refresh: () => void
  setPage: (page: number) => void
  retry: (id: string) => Promise<void>
  abandon: (id: string) => Promise<void>
  abandonAll: () => Promise<number>
}

// Defaults to the (server-side default) pending_republish list -- what an
// operator triages after a NATS outage. Mutations refetch the current page
// rather than patching the array locally -- with server-side pagination and
// a total count, splicing in place would drift from what the server holds
// (same convention as pms/hooks/useProducts).
export function useWebhookFallbackEvents(token: string): UseWebhookFallbackEventsState & UseWebhookFallbackEventsActions {
  const [events, setEvents] = useState<WebhookFallbackEvent[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const result = await api.listWebhookFallbackEvents(token, 'pending_republish', page, PER_PAGE)
      setEvents(result.events)
      setTotal(result.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('integrations:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token, page])

  useEffect(() => { load() }, [load])

  const retry = useCallback(async (id: string): Promise<void> => {
    await api.retryWebhookFallbackEvent(token, id)
    await load()
  }, [token, load])

  const abandon = useCallback(async (id: string): Promise<void> => {
    await api.abandonWebhookFallbackEvent(token, id)
    await load()
  }, [token, load])

  // Returns how many rows were abandoned, for a confirmation toast --
  // resets to page 1 since every currently-pending row is now gone.
  const abandonAll = useCallback(async (): Promise<number> => {
    const { abandoned } = await api.abandonAllWebhookFallbackEvents(token, 'pending_republish')
    if (page !== 1) {
      setPage(1)
    } else {
      await load()
    }
    return abandoned
  }, [token, page, load])

  return { events, total, page, perPage: PER_PAGE, loading, error, refresh: load, setPage, retry, abandon, abandonAll }
}
