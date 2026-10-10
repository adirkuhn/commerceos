import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { SyncLogEntry } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseSyncLogsState {
  entries: SyncLogEntry[]
  total: number
  loading: boolean
  error: string | null
}

interface UseSyncLogsActions {
  refresh: () => void
}

export function useSyncLogs(token: string, params: { page?: number; perPage?: number } = {}): UseSyncLogsState & UseSyncLogsActions {
  const { page, perPage } = params
  const [entries, setEntries] = useState<SyncLogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const result = await api.listAllSyncLogs(token, { page, perPage })
      setEntries(result.entries)
      setTotal(result.total)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('integrations:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token, page, perPage])

  useEffect(() => { load() }, [load])

  return { entries, total, loading, error, refresh: load }
}
