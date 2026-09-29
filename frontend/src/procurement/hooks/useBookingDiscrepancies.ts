import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { BookingDiscrepancyReport } from '../types'
import * as api from '../api'

interface UseBookingDiscrepanciesState {
  report: BookingDiscrepancyReport
  loading: boolean
  error: string | null
}

const EMPTY_REPORT: BookingDiscrepancyReport = { missing_items: [], damaged_lost: [] }

// Tenant-wide report of every completed booking's under-scanned line items
// and damaged/lost stock -- derived on demand server-side, not persisted.
export function useBookingDiscrepancies(token: string): UseBookingDiscrepanciesState & { refresh: () => void } {
  const [report, setReport] = useState<BookingDiscrepancyReport>(EMPTY_REPORT)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listBookingDiscrepancies(token)
      setReport(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('procurement:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  return { report, loading, error, refresh: load }
}
