import { useState } from 'react'
import i18next from 'i18next'
import { isAuthError } from '../auth/isAuthError'
import type { StockShortfall, StockShortfallResolution } from './types'
import { apiErrorText } from '../api/apiErrorText'

interface CompletionResult {
  completed: boolean
  stock_shortfalls?: StockShortfall[]
}

// Finishing a picking or packing session: the review step, the complete
// call, and the stock-shortfall round trip when it comes back
// completed:false (see StockShortfallDialog).
export function useSessionCompletion<R extends CompletionResult>(
  complete: (resolutions?: Record<string, StockShortfallResolution>) => Promise<R>,
  onAuthError: () => void,
  setError: (message: string) => void,
) {
  const [finishing, setFinishing] = useState(false)
  const [confirmingFinish, setConfirmingFinish] = useState(false)
  const [result, setResult] = useState<R | null>(null)
  const [stockShortfalls, setStockShortfalls] = useState<StockShortfall[] | null>(null)

  // resolutions is only passed when resubmitting from StockShortfallDialog.
  async function confirmComplete(resolutions?: Record<string, StockShortfallResolution>) {
    setFinishing(true)
    try {
      const r = await complete(resolutions)
      if (!r.completed) {
        setStockShortfalls(r.stock_shortfalls ?? [])
        return
      }
      setStockShortfalls(null)
      setResult(r)
      setConfirmingFinish(false)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(apiErrorText(err) ?? i18next.t('scanner:errors.complete'))
    } finally {
      setFinishing(false)
    }
  }

  return { finishing, confirmingFinish, setConfirmingFinish, result, stockShortfalls, setStockShortfalls, confirmComplete }
}
