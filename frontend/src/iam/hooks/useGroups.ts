import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { Group, CreateGroupPayload, UpdateGroupPayload } from '../types'
import * as api from '../api'

interface UseGroupsState {
  groups: Group[]
  loading: boolean
  error: string | null
}

interface UseGroupsActions {
  refresh: () => void
  create: (payload: CreateGroupPayload) => Promise<Group>
  update: (id: string, payload: UpdateGroupPayload) => Promise<Group>
  remove: (id: string) => Promise<void>
}

export function useGroups(token: string): UseGroupsState & UseGroupsActions {
  const [groups, setGroups] = useState<Group[]>([])
  // true from the first render when a load is coming -- see useConnections.
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listGroups(token)
      setGroups(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('iam:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (payload: CreateGroupPayload): Promise<Group> => {
    const group = await api.createGroup(token, payload)
    setGroups(prev => [...prev, group])
    return group
  }, [token])

  const update = useCallback(async (id: string, payload: UpdateGroupPayload): Promise<Group> => {
    const updated = await api.updateGroup(token, id, payload)
    setGroups(prev => prev.map(g => g.id === id ? updated : g))
    return updated
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteGroup(token, id)
    setGroups(prev => prev.filter(g => g.id !== id))
  }, [token])

  return { groups, loading, error, refresh: load, create, update, remove }
}
