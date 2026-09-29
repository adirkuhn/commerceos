import i18next from 'i18next'
import { useState, useCallback } from 'react'
import type { DiscardBatchInput, DiscardRecord } from '../types'
import * as api from '../api'

// Mutation-only -- shared by every list that can discard a row (the
// expired-review queue, the "expiring soon" report, and the scanner's
// Discard app), each of which refreshes its own list afterwards rather
// than this hook owning any list state itself.
export function useDiscardBatch(token: string): {
  discarding: boolean
  error: string | null
  discard: (skuId: string, input: DiscardBatchInput) => Promise<DiscardRecord>
} {
  const [discarding, setDiscarding] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const discard = useCallback(async (skuId: string, input: DiscardBatchInput): Promise<DiscardRecord> => {
    setDiscarding(true)
    setError(null)
    try {
      return await api.discardBatch(token, skuId, input)
    } catch (e) {
      const message = e instanceof Error ? e.message : i18next.t('inventory:errors.unknown')
      setError(message)
      throw e
    } finally {
      setDiscarding(false)
    }
  }, [token])

  return { discarding, error, discard }
}
