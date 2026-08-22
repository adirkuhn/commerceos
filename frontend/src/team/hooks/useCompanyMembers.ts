import { useState, useEffect, useCallback } from 'react'
import type { CompanyMember, MemberRole } from '../types'
import * as api from '../api'

interface UseCompanyMembersState {
  members: CompanyMember[]
  loading: boolean
  error: string | null
}

interface UseCompanyMembersActions {
  refresh: () => void
  updateRole: (membershipId: string, role: MemberRole) => Promise<void>
  remove: (membershipId: string) => Promise<void>
}

export function useCompanyMembers(token: string): UseCompanyMembersState & UseCompanyMembersActions {
  const [members, setMembers] = useState<CompanyMember[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      setMembers(await api.listMembers(token))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const updateRole = useCallback(async (membershipId: string, role: MemberRole) => {
    await api.updateMemberRole(token, membershipId, role)
    await load()
  }, [token, load])

  const remove = useCallback(async (membershipId: string) => {
    await api.removeMember(token, membershipId)
    await load()
  }, [token, load])

  return { members, loading, error, refresh: load, updateRole, remove }
}
