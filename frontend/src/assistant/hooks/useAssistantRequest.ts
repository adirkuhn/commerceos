import i18next from 'i18next'
import { useCallback, useEffect, useRef, useState } from 'react'
import { apiErrorText } from '../../api/apiErrorText'
import { getAssistantRequest } from '../api'
import { isWorking, POLL_INTERVAL_MS } from '../format'
import type { AssistantRequest, AssistantRequestDetail } from '../types'

export interface AssistantRequestState {
  data: AssistantRequestDetail | null
  loading: boolean
  error: string | null
  refresh: () => void
  // Shows a request returned by reply/cancel right away (and restarts
  // polling when it's queued again) without waiting for the next read.
  setRequest: (request: AssistantRequest) => void
}

// One request with its conversation, re-read every POLL_INTERVAL_MS while
// the model may still be working on it (queued/processing).
export function useAssistantRequest(token: string, id: string): AssistantRequestState {
  const [data, setData] = useState<AssistantRequestDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const latest = useRef(0)

  const load = useCallback(async () => {
    if (!token) return
    const call = ++latest.current
    try {
      const out = await getAssistantRequest(token, id)
      if (call !== latest.current) return
      setData(out)
      setError(null)
    } catch (e) {
      if (call === latest.current) setError(apiErrorText(e) ?? i18next.t('ui:errors.load'))
    } finally {
      if (call === latest.current) setLoading(false)
    }
  }, [token, id])

  useEffect(() => {
    setData(null)
    setLoading(true)
    setError(null)
    load()
  }, [load])

  const status = data?.request.status
  useEffect(() => {
    if (!status || !isWorking(status)) return
    const timer = setTimeout(() => { load() }, POLL_INTERVAL_MS)
    return () => clearTimeout(timer)
  }, [status, data, load])

  const setRequest = useCallback((request: AssistantRequest) => {
    setData(d => (d ? { ...d, request } : d))
  }, [])

  return { data, loading, error, refresh: load, setRequest }
}
