import { useCallback, useEffect, useState } from 'react'
import { getNotificationPreferences, updateNotificationPreferences } from './api'
import type { NotificationPreferences, NotificationPreferencesInput } from './types'

export function useNotificationPreferences(token: string) {
  const [preferences, setPreferences] = useState<NotificationPreferences | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    getNotificationPreferences(token)
      .then(p => { if (!cancelled) setPreferences(p) })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load notification settings') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token])

  // Rejects with NotificationPreferencesError so the form can place it.
  const save = useCallback(async (input: NotificationPreferencesInput) => {
    const saved = await updateNotificationPreferences(token, input)
    setPreferences(saved)
    return saved
  }, [token])

  return { preferences, loading, error, save }
}
