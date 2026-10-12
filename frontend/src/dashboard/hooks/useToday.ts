import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import * as dashboardApi from '../api'
import type { DashboardStatsReport, TodayQueue } from '../types'
import { apiErrorText } from '../../api/apiErrorText'

interface TodayState {
  queue: TodayQueue | null
  report: DashboardStatsReport | null
  loading: boolean
  error: string | null
  refresh: () => void
}

// The Today page's data: the work-queue counts and this month's figures,
// loaded together.
export function useToday(token: string): TodayState {
  const [queue, setQueue] = useState<TodayQueue | null>(null)
  const [report, setReport] = useState<DashboardStatsReport | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const [q, r] = await Promise.all([dashboardApi.getTodayQueue(token), dashboardApi.getDashboardStats(token)])
      setQueue(q)
      setReport(r)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('dashboard:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  return { queue, report, loading, error, refresh: load }
}
