import { useState, useEffect } from 'react'
import { dateRange, type ReportsPeriod } from '../period'

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
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Unknown error') })
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

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError(null)
    const { from, to } = dateRange(period, new Date())
    fetchFn(token, from, to)
      .then(result => { if (!cancelled) setData(result) })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, period, fetchFn])

  return { data, loading, error }
}
