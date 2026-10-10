import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { SalesOrderActivity } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

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
      setError(apiErrorText(e) ?? i18next.t('orders:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token, salesOrderId])

  useEffect(() => { load() }, [load])

  return { activities, loading, error, refresh: load }
}
