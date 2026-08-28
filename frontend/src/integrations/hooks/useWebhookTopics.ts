import { useState, useEffect } from 'react'
import * as api from '../api'

// Provider -> topics this tenant could actually write a fan-out rule for --
// see api.listAvailableWebhookTopics's doc comment. Always loaded (not
// gated by tab) since the Fan-out form's Topic dropdown needs it
// immediately; the request is cheap (one DB read + already-cached provider
// subscription calls).
export function useWebhookTopics(token: string): { topicsByProvider: Record<string, string[]>; loading: boolean } {
  const [topicsByProvider, setTopicsByProvider] = useState<Record<string, string[]>>({})
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    api.listAvailableWebhookTopics(token)
      .then(result => { if (!cancelled) setTopicsByProvider(result) })
      .catch(() => { /* best-effort -- form still works with an empty list */ })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token])

  return { topicsByProvider, loading }
}
