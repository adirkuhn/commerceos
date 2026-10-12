import i18next from 'i18next'
import { useEffect, useState } from 'react'
import * as dashboardApi from '../api'
import type { SalesComparison, SalesPeriod } from '../types'
import { apiErrorText } from '../../api/apiErrorText'

interface SalesComparisonState {
  data: SalesComparison | null
  loading: boolean
  error: string | null
}

// The dashboard sales card's figures for one period. Switching period
// quickly never lets an older, slower answer overwrite the newer one.
export function useSalesComparison(token: string, period: SalesPeriod): SalesComparisonState {
  const [data, setData] = useState<SalesComparison | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    let current = true
    setLoading(true)
    setError(null)
    dashboardApi.getSalesComparison(token, period)
      .then(c => { if (current) setData(c) })
      .catch(e => {
        if (!current) return
        setData(null)
        setError(apiErrorText(e) ?? i18next.t('dashboard:errors.unknown'))
      })
      .finally(() => { if (current) setLoading(false) })
    return () => { current = false }
  }, [token, period])

  return { data, loading, error }
}
