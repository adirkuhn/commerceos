import { useState, useEffect, useCallback } from 'react'
import i18next from 'i18next'
import type { TenantSettings, TenantSettingsInput } from '../types'
import * as api from '../api'

interface UseTenantSettingsState {
  settings: TenantSettings | null
  loading: boolean
  error: string | null
}

interface UseTenantSettingsActions {
  refresh: () => void
  save: (input: TenantSettingsInput) => Promise<TenantSettings>
}

export function useTenantSettings(token: string): UseTenantSettingsState & UseTenantSettingsActions {
  const [settings, setSettings] = useState<TenantSettings | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.getTenantSettings(token)
      setSettings(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('settings:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const save = useCallback(async (input: TenantSettingsInput): Promise<TenantSettings> => {
    const result = await api.updateTenantSettings(token, input)
    setSettings(result)
    return result
  }, [token])

  return { settings, loading, error, refresh: load, save }
}
