import i18next from 'i18next'
import { useState, useEffect } from 'react'
import { dateRange, type ReportsPeriod } from '../period'
import { useDateFormat } from '../../format/DateFormatProvider'
import { apiErrorText } from '../../api/apiErrorText'

// useReport fetches one report for an explicit [from, to) range -- the
// building block behind useSingleReport below. Pages with their own range
// control (e.g. a custom date picker, not just a period preset) use this
// directly; from/to must be stable across renders (e.g. computed with
// useMemo) or this refetches on every render.
export function useReport<T>(
  token: string,
  from: string,
  to: string,
  fetchFn: (token: string, from: string, to: string) => Promise<T>,
): { data: T | null; loading: boolean; error: string | null } {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError(null)
    fetchFn(token, from, to)
      .then(result => { if (!cancelled) setData(result) })
      .catch(e => { if (!cancelled) setError(apiErrorText(e) ?? i18next.t('ui:errors.unknown')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, from, to, fetchFn])

  return { data, loading, error }
}

// useSingleReport backs each standalone report page: one report, one
// period selector, fetched independently of every other report -- unlike
// the old all-in-one Reports page, navigating between report pages here
// only ever fetches the report actually being viewed.
export function useSingleReport<T>(
  token: string,
  period: ReportsPeriod,
  fetchFn: (token: string, from: string, to: string) => Promise<T>,
): { data: T | null; loading: boolean; error: string | null } {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // The range is whole days in the tenant's timezone -- the same days the
  // page's subtitle shows and the backend buckets by.
  const { timeZone } = useDateFormat().settings

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError(null)
    const { from, to } = dateRange(period, new Date(), timeZone)
    fetchFn(token, from, to)
      .then(result => { if (!cancelled) setData(result) })
      .catch(e => { if (!cancelled) setError(apiErrorText(e) ?? i18next.t('ui:errors.unknown')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, period, timeZone, fetchFn])

  return { data, loading, error }
}
