import { useState, useEffect } from 'react'
import * as api from '../api'
import type { AvailableWebhookTopics } from '../api'

const EMPTY: AvailableWebhookTopics = { all_events: {}, active_webhooks: {} }

// See api.AvailableWebhookTopics' field docs -- allEvents is the reference
// catalog, activeWebhooks is what's actually subscribed right now. Always
// loaded (not gated by tab) since the Fan-out form's Topic dropdown needs
// it immediately; the request is cheap (one DB read + already-cached
// provider subscription calls).
export function useWebhookTopics(token: string): { topics: AvailableWebhookTopics; loading: boolean } {
  const [topics, setTopics] = useState<AvailableWebhookTopics>(EMPTY)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    api.listAvailableWebhookTopics(token)
      .then(result => { if (!cancelled) setTopics(result) })
      .catch(() => { /* best-effort -- form still works with an empty list */ })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token])

  return { topics, loading }
}
