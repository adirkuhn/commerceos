import i18next from 'i18next'
import { useState } from 'react'
import * as packingApi from '../scanner/api'
import type { PackingSessionDetail, CompletePackingResult, StockShortfall, StockShortfallResolution } from '../scanner/types'
import { apiErrorText } from '../api/apiErrorText'

interface PackingFinishOptions {
  token: string | null
  session: PackingSessionDetail | null
  loadSession: (sessionId: string) => Promise<void>
  setError: (message: string) => void
  // Closes the line/add-item side panels (a reset discards what they show).
  closePanels: () => void
  onCompleted: () => Promise<void>
}

// Ending a packing session: the review-then-complete step (with its stock
// shortfall round trip) and the "start over" reset.
export function usePackingFinish({ token, session, loadSession, setError, closePanels, onCompleted }: PackingFinishOptions) {
  const [resetConfirmOpen, setResetConfirmOpen] = useState(false)
  const [resetting, setResetting] = useState(false)

  const [completing, setCompleting] = useState(false)
  const [completeResult, setCompleteResult] = useState<CompletePackingResult | null>(null)
  const [confirmingFinish, setConfirmingFinish] = useState(false)
  // Set when completePacking comes back completed:false -- a substitute or
  // extra item is short on tracked stock; staff decide per item and finish
  // again (see StockShortfallDialog).
  const [stockShortfalls, setStockShortfalls] = useState<StockShortfall[] | null>(null)

  // Discards every pack scan on the session and hands it back fresh -- the
  // "start over" escape hatch for switching boxes or a mis-scan spree. Only
  // legal pre-complete (see ResetPackingSessionUseCase's doc).
  async function resetSession() {
    if (!session || !token) return
    setResetting(true)
    setError('')
    try {
      await packingApi.resetPackingSession(token, session.session.id)
      setResetConfirmOpen(false)
      closePanels()
      await loadSession(session.session.id)
    } catch (err) {
      setError(apiErrorText(err) ?? i18next.t('packing:errors.reset'))
    } finally {
      setResetting(false)
    }
  }

  // resolutions is only passed when resubmitting from StockShortfallDialog.
  async function complete(resolutions?: Record<string, StockShortfallResolution>) {
    if (!session || !token) return
    setCompleting(true)
    setError('')
    try {
      const result = await packingApi.completePacking(token, session.session.id, resolutions)
      if (!result.completed) {
        setStockShortfalls(result.stock_shortfalls ?? [])
        return
      }
      setStockShortfalls(null)
      setCompleteResult(result)
      setConfirmingFinish(false)
      await onCompleted()
    } catch (err) {
      setError(apiErrorText(err) ?? i18next.t('packing:errors.complete'))
    } finally {
      setCompleting(false)
    }
  }

  // A different order (or none) is on screen now.
  function clear() {
    setCompleteResult(null)
    setConfirmingFinish(false)
  }

  return {
    resetConfirmOpen, openResetConfirm: () => setResetConfirmOpen(true), closeResetConfirm: () => setResetConfirmOpen(false),
    resetting, resetSession,
    confirmingFinish, startFinish: () => setConfirmingFinish(true), cancelFinish: () => setConfirmingFinish(false),
    completing, completeResult, complete, clearResult: () => setCompleteResult(null),
    stockShortfalls, cancelShortfalls: () => setStockShortfalls(null),
    clear,
  }
}
