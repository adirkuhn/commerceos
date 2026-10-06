import i18next from 'i18next'
import { useCallback, useEffect, useRef, useState } from 'react'
import * as api from '../api'
import type { ReorderRecommendations, ReorderRecommendationsQuery } from '../types'

interface State {
  data: ReorderRecommendations | null
  loading: boolean
  error: string | null
  refresh: () => void
}

// Recommendations for the planning window; a newer query always wins over
// a slower answer to an older one.
export function useReorderRecommendations(token: string, query: ReorderRecommendationsQuery): State {
  const [data, setData] = useState<ReorderRecommendations | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const latest = useRef(0)
  const { coverage_days, safety_days, supplier_id } = query

  const load = useCallback(async () => {
    if (!token) return
    const call = ++latest.current
    setLoading(true)
    setError(null)
    try {
      const out = await api.getReorderRecommendations(token, { coverage_days, safety_days, supplier_id })
      if (call === latest.current) setData(out)
    } catch (e) {
      if (call === latest.current) {
        setData(null)
        setError(e instanceof Error ? e.message : i18next.t('purchasePlanning:errors.unknown'))
      }
    } finally {
      if (call === latest.current) setLoading(false)
    }
  }, [token, coverage_days, safety_days, supplier_id])

  useEffect(() => { load() }, [load])

  return { data, loading, error, refresh: load }
}
