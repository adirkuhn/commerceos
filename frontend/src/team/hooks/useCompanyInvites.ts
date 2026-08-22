import { useState, useEffect, useCallback } from 'react'
import type { CompanyInvite, MemberRole } from '../types'
import * as api from '../api'

interface UseCompanyInvitesState {
  invites: CompanyInvite[]
  loading: boolean
  error: string | null
}

interface UseCompanyInvitesActions {
  refresh: () => void
  create: (email: string, role: MemberRole) => Promise<void>
  revoke: (inviteId: string) => Promise<void>
}

export function useCompanyInvites(token: string): UseCompanyInvitesState & UseCompanyInvitesActions {
  const [invites, setInvites] = useState<CompanyInvite[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      setInvites(await api.listInvites(token))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (email: string, role: MemberRole) => {
    await api.createInvite(token, email, role)
    await load()
  }, [token, load])

  const revoke = useCallback(async (inviteId: string) => {
    await api.revokeInvite(token, inviteId)
    await load()
  }, [token, load])

  return { invites, loading, error, refresh: load, create, revoke }
}
