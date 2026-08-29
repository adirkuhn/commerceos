import { useState, useEffect, useCallback } from 'react'
import type { WebhookEvent } from '../types'
import * as api from '../api'

interface UseWebhookEventsState {
  events: WebhookEvent[]
  loading: boolean
  error: string | null
}

interface UseWebhookEventsActions {
  refresh: () => void
}

// Backs the Webhooks page's Messages tab -- every inbound webhook this
// tenant has received, optionally filtered to one sender. Refetches
// whenever provider changes. See backend/domain/webhooks.WebhookEvent.
export function useWebhookEvents(token: string, provider: string): UseWebhookEventsState & UseWebhookEventsActions {
  const [events, setEvents] = useState<WebhookEvent[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const result = await api.listWebhookEvents(token, { provider: provider || undefined })
      setEvents(result)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, provider])

  useEffect(() => { load() }, [load])

  return { events, loading, error, refresh: load }
}
