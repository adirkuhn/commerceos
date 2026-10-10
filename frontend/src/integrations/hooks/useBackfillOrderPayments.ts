import i18next from 'i18next'
import { useCallback, useState } from 'react'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

export interface BackfillProgress {
  checked: number
  updated: number
  errors: string[]
}

const EMPTY: BackfillProgress = { checked: 0, updated: 0, errors: [] }

// Drives POST .../backfill-order-payments batch by batch (the backend
// checks 25 orders per call) until next_cursor comes back null, keeping a
// running total so the page can show progress while it works.
export function useBackfillOrderPayments(token: string, connectionId: string) {
  const [progress, setProgress] = useState<BackfillProgress>(EMPTY)
  const [running, setRunning] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState('')

  const run = useCallback(async () => {
    setRunning(true)
    setDone(false)
    setError('')
    setProgress(EMPTY)
    let cursor: string | null = null
    try {
      do {
        const batch = await api.backfillWooOrderPayments(token, connectionId, cursor)
        setProgress(p => ({ checked: p.checked + batch.checked, updated: p.updated + batch.updated, errors: [...p.errors, ...batch.errors] }))
        cursor = batch.next_cursor
      } while (cursor)
      setDone(true)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('integrations:woo.backfillPaymentsFailed'))
    } finally {
      setRunning(false)
    }
  }, [token, connectionId])

  return { run, progress, running, done, error }
}
