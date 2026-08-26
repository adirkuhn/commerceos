import { useState, useEffect, useCallback } from 'react'
import type { SalesOrderActivity } from '../types'
import * as api from '../api'

interface UseSalesOrderActivitiesState {
  activities: SalesOrderActivity[]
  loading: boolean
  error: string | null
}

export function useSalesOrderActivities(token: string, salesOrderId: string | null): UseSalesOrderActivitiesState & { refresh: () => void } {
  const [activities, setActivities] = useState<SalesOrderActivity[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !salesOrderId) {
      setActivities([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      const result = await api.listSalesOrderActivities(token, salesOrderId)
      setActivities(result.activities)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, salesOrderId])

  useEffect(() => { load() }, [load])

  return { activities, loading, error, refresh: load }
}
