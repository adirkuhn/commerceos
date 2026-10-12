import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import { apiErrorText } from '../../api/apiErrorText'
import { useQueryEnum, useQueryPage, useQueryString } from '../../routing/useQueryState'
import * as api from './api'
import type { CloseToExpiryBatch, CloseToExpiryList } from './types'

// The close-to-expiry tab's state: which window (?window=, default the
// largest), whether it shows batches already marked as seen (?seen=1) and
// the page (?page=) -- all in the URL so refresh and Back keep the view.
export function useCloseToExpiry(token: string) {
  const [windowParam, writeWindow] = useQueryString('window', { resets: ['page'] })
  const [list, writeList] = useQueryEnum('seen', ['1', '0'] as const, '0', { resets: ['page'] })
  const [page, setPage] = useQueryPage()
  const windowDays = windowParam !== null && /^\d+$/.test(windowParam) ? Number(windowParam) : undefined
  const seen = list === '1'

  const [data, setData] = useState<CloseToExpiryList | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [actingId, setActingId] = useState<string | null>(null)
  const [actionError, setActionError] = useState('')

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      setData(await api.listCloseToExpiry(token, { window: windowDays, acknowledged: seen, page }))
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('inventory:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token, windowDays, seen, page])

  useEffect(() => { load() }, [load])

  // Runs a mark/undo for the listed window, then reloads the counts and list.
  const act = useCallback(async (batch: CloseToExpiryBatch, run: (windowDays: number) => Promise<void>) => {
    if (!data) return
    setActionError('')
    setActingId(batch.batch_id)
    try {
      await run(data.window.window_days)
      await load()
    } catch (e) {
      setActionError(apiErrorText(e) ?? i18next.t('inventory:errors.unknown'))
    } finally {
      setActingId(null)
    }
  }, [data, load])

  const acknowledge = useCallback((batch: CloseToExpiryBatch) =>
    act(batch, w => api.acknowledgeBatchExpiry(token, batch.batch_id, w)), [act, token])
  const undo = useCallback((batch: CloseToExpiryBatch) =>
    act(batch, w => api.unacknowledgeBatchExpiry(token, batch.batch_id, w)), [act, token])

  return {
    data, loading, error, actingId, actionError, seen, page, setPage,
    setWindow: useCallback((days: number) => writeWindow(String(days)), [writeWindow]),
    setSeen: useCallback((on: boolean) => writeList(on ? '1' : '0'), [writeList]),
    acknowledge, undo, refresh: load,
  }
}

export type CloseToExpiryState = ReturnType<typeof useCloseToExpiry>
