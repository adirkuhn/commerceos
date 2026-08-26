import { useState, useEffect, useCallback } from 'react'
import * as api from '../api'
import type { UserActivity } from '../types'

interface UseUserActivitiesState {
  activities: UserActivity[]
  loading: boolean
  error: string | null
}

export function useUserActivities(token: string, userId: string): UseUserActivitiesState & { refresh: () => void } {
  const [activities, setActivities] = useState<UserActivity[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.getUserActivities(token, userId)
      setActivities(data.activities)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, userId])

  useEffect(() => { load() }, [load])

  return { activities, loading, error, refresh: load }
}
