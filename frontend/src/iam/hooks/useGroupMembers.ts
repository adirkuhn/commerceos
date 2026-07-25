import { useState, useEffect, useCallback } from 'react'
import * as api from '../api'

interface UseGroupMembersState {
  members: string[]
  loading: boolean
  error: string | null
}

interface UseGroupMembersActions {
  refresh: () => void
  add: (userId: string) => Promise<void>
  remove: (userId: string) => Promise<void>
}

export function useGroupMembers(token: string, groupId: string): UseGroupMembersState & UseGroupMembersActions {
  const [members, setMembers] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.listMembers(token, groupId)
      setMembers(data.members)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, groupId])

  useEffect(() => { load() }, [load])

  const add = useCallback(async (userId: string): Promise<void> => {
    await api.addMember(token, groupId, { user_id: userId })
    setMembers(prev => [...prev, userId])
  }, [token, groupId])

  const remove = useCallback(async (userId: string): Promise<void> => {
    await api.removeMember(token, groupId, userId)
    setMembers(prev => prev.filter(id => id !== userId))
  }, [token, groupId])

  return { members, loading, error, refresh: load, add, remove }
}
