import i18next from 'i18next'
import { useCallback, useEffect, useRef, useState } from 'react'
import { apiErrorText } from '../../api/apiErrorText'

export interface RequestState<T> {
  data: T | null
  loading: boolean
  error: string | null
  refresh: () => void
}

// Runs fetch whenever key changes (key = the request's identity, e.g. its
// serialized query); a newer request always wins over a slower answer to
// an older one. Nothing runs without a token or while disabled.
export function useLatestRequest<T>(token: string, key: string, fetch: (token: string) => Promise<T>, enabled = true): RequestState<T> {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const latest = useRef(0)
  const fetchRef = useRef(fetch)
  fetchRef.current = fetch

  const load = useCallback(async () => {
    if (!token || !enabled) return
    const call = ++latest.current
    setLoading(true)
    setError(null)
    try {
      const out = await fetchRef.current(token)
      if (call === latest.current) setData(out)
    } catch (e) {
      if (call === latest.current) {
        setData(null)
        setError(apiErrorText(e) ?? i18next.t('purchasePlanning:errors.unknown'))
      }
    } finally {
      if (call === latest.current) setLoading(false)
    }
    // key stands in for fetch's inputs.
  }, [token, key, enabled]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => { load() }, [load])

  return { data, loading, error, refresh: load }
}
