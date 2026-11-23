import i18next from 'i18next'
import { useState, useEffect, useCallback, useRef } from 'react'
import type { AIConnection } from '../types'
import * as api from '../api'
import type { AIConnectionPayload, AIConnectionPatch } from '../api'
import { apiErrorText } from '../../api/apiErrorText'

// The tenant's ordered AI connections. create/update/remove reject so the
// caller can show the error where the user acted; a failed reorder (a drag)
// rolls back and lands in `error`.
export function useAIConnections(token: string) {
  const [connections, setConnections] = useState<AIConnection[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const latest = useRef<AIConnection[]>([])
  latest.current = connections

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      setConnections(await api.listAIConnections(token))
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('integrations:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (p: AIConnectionPayload): Promise<AIConnection> => {
    const created = await api.createAIConnection(token, p)
    setConnections(cs => [...cs, created])
    return created
  }, [token])

  const update = useCallback(async (id: string, p: AIConnectionPatch): Promise<AIConnection> => {
    const updated = await api.updateAIConnection(token, id, p)
    setConnections(cs => cs.map(c => (c.id === id ? updated : c)))
    return updated
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteAIConnection(token, id)
    setConnections(await api.listAIConnections(token))
  }, [token])

  const reorder = useCallback(async (ids: string[]): Promise<void> => {
    const previous = latest.current
    const byId = new Map(previous.map(c => [c.id, c]))
    setConnections(ids.flatMap(id => byId.get(id) ?? []))
    setError(null)
    try {
      setConnections(await api.reorderAIConnections(token, ids))
    } catch (e) {
      setConnections(previous)
      setError(apiErrorText(e) ?? i18next.t('integrations:unknownError'))
    }
  }, [token])

  return { connections, loading, error, refresh: load, create, update, remove, reorder }
}
