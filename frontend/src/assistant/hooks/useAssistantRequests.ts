import i18next from 'i18next'
import { useCallback, useEffect, useRef, useState } from 'react'
import { apiErrorText } from '../../api/apiErrorText'
import { listAssistantRequests } from '../api'
import type { AssistantListQuery, AssistantRequest, AssistantRequestStatus } from '../types'

export interface AssistantRequestsState {
  items: AssistantRequest[]
  loading: boolean
  error: string | null
  hasMore: boolean
  loadMore: () => void
  refresh: () => void
  replace: (request: AssistantRequest) => void
  // Shows a request in the list: a new chat goes on top, a known one is
  // updated in place.
  upsert: (request: AssistantRequest) => void
}

// The tenant's requests, newest first, optionally one status; "load more"
// appends the next cursor page. A new filter starts over.
export function useAssistantRequests(token: string, status: AssistantRequestStatus | undefined): AssistantRequestsState {
  const [items, setItems] = useState<AssistantRequest[]>([])
  const [cursor, setCursor] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const latest = useRef(0)

  const fetchPage = useCallback(async (after: string | null) => {
    if (!token) return
    const call = ++latest.current
    setLoading(true)
    setError(null)
    try {
      const query: AssistantListQuery = status ? { status } : {}
      if (after) query.cursor = after
      const page = await listAssistantRequests(token, query)
      if (call !== latest.current) return
      setItems(prev => (after ? [...prev, ...page.requests] : page.requests))
      setCursor(page.next_cursor ?? null)
    } catch (e) {
      if (call === latest.current) setError(apiErrorText(e) ?? i18next.t('ui:errors.load'))
    } finally {
      if (call === latest.current) setLoading(false)
    }
  }, [token, status])

  useEffect(() => {
    setItems([])
    setCursor(null)
    fetchPage(null)
  }, [fetchPage])

  const loadMore = useCallback(() => { if (cursor) fetchPage(cursor) }, [cursor, fetchPage])
  const refresh = useCallback(() => { fetchPage(null) }, [fetchPage])
  const replace = useCallback((request: AssistantRequest) => {
    setItems(prev => prev.map(r => (r.id === request.id ? request : r)))
  }, [])

  const upsert = useCallback((request: AssistantRequest) => {
    setItems(prev => (prev.some(r => r.id === request.id)
      ? prev.map(r => (r.id === request.id ? request : r))
      : [request, ...prev]))
  }, [])

  return { items, loading, error, hasMore: cursor !== null, loadMore, refresh, replace, upsert }
}
