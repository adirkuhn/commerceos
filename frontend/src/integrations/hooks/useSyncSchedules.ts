import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { SyncSchedule } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseSyncSchedulesState {
  schedules: SyncSchedule[]
  loading: boolean
  error: string | null
}

interface UseSyncSchedulesActions {
  refresh: () => void
  save: (schedules: SyncSchedule[]) => Promise<SyncSchedule[]>
}

// connectionId is null while no connection is selected — nothing to load
// until the user opens a specific service's settings page.
export function useSyncSchedules(token: string, connectionId: string | null): UseSyncSchedulesState & UseSyncSchedulesActions {
  const [schedules, setSchedules] = useState<SyncSchedule[]>([])
  // Starts true whenever there's a connection to fetch for — callers that
  // gate rendering on `loading` (e.g. hydrate-once patterns) would
  // otherwise see a false "not loading" on the very first render, before
  // the mount effect below has had a chance to flip it.
  const [loading, setLoading] = useState(!!connectionId)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !connectionId) {
      setSchedules([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      const data = await api.listSyncSchedules(token, connectionId)
      setSchedules(data)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('integrations:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token, connectionId])

  useEffect(() => { load() }, [load])

  const save = useCallback(async (updated: SyncSchedule[]): Promise<SyncSchedule[]> => {
    if (!connectionId) throw new Error(i18next.t('integrations:noConnectionSelected'))
    const result = await api.upsertSyncSchedules(token, connectionId, updated)
    setSchedules(result)
    return result
  }, [token, connectionId])

  return { schedules, loading, error, refresh: load, save }
}
