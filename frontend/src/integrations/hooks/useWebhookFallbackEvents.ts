import { useState, useEffect, useCallback } from 'react'
import type { WebhookFallbackEvent } from '../types'
import * as api from '../api'

interface UseWebhookFallbackEventsState {
  events: WebhookFallbackEvent[]
  loading: boolean
  error: string | null
}

interface UseWebhookFallbackEventsActions {
  refresh: () => void
  retry: (id: string) => Promise<void>
  abandon: (id: string) => Promise<void>
}

// Defaults to the (server-side default) pending_republish list -- what an
// operator triages after a NATS outage. retry/abandon both drop the row
// out of this list on success (retried -> republished, abandoned -> no
// longer pending), matching the backend's status transitions.
export function useWebhookFallbackEvents(token: string): UseWebhookFallbackEventsState & UseWebhookFallbackEventsActions {
  const [events, setEvents] = useState<WebhookFallbackEvent[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listWebhookFallbackEvents(token, 'pending_republish')
      setEvents(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const retry = useCallback(async (id: string): Promise<void> => {
    await api.retryWebhookFallbackEvent(token, id)
    setEvents(prev => prev.filter(e => e.id !== id))
  }, [token])

  const abandon = useCallback(async (id: string): Promise<void> => {
    await api.abandonWebhookFallbackEvent(token, id)
    setEvents(prev => prev.filter(e => e.id !== id))
  }, [token])

  return { events, loading, error, refresh: load, retry, abandon }
}
