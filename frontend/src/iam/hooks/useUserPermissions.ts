import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseUserPermissionsState {
  permissions: string[]
  loading: boolean
  error: string | null
}

export function useUserPermissions(token: string, userId: string): UseUserPermissionsState & { refresh: () => void } {
  const [permissions, setPermissions] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.getUserEffectivePermissions(token, userId)
      setPermissions(data.permissions)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('iam:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token, userId])

  useEffect(() => { load() }, [load])

  return { permissions, loading, error, refresh: load }
}
