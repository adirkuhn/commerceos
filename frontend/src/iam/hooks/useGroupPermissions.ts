import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseGroupPermissionsState {
  permissions: string[]
  loading: boolean
  error: string | null
}

interface UseGroupPermissionsActions {
  refresh: () => void
  assign: (rule: string) => Promise<void>
  revoke: (rule: string) => Promise<void>
}

export function useGroupPermissions(token: string, groupId: string): UseGroupPermissionsState & UseGroupPermissionsActions {
  const [permissions, setPermissions] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.listGroupPermissions(token, groupId)
      setPermissions(data.permissions)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('iam:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token, groupId])

  useEffect(() => { load() }, [load])

  const assign = useCallback(async (rule: string): Promise<void> => {
    await api.assignPermission(token, groupId, { rule })
    setPermissions(prev => [...new Set([...prev, rule])])
  }, [token, groupId])

  const revoke = useCallback(async (rule: string): Promise<void> => {
    await api.revokePermission(token, groupId, rule)
    setPermissions(prev => prev.filter(r => r !== rule))
  }, [token, groupId])

  return { permissions, loading, error, refresh: load, assign, revoke }
}
