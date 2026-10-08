import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import { listBolShipmentConfirmations, isBolConfirmationPending } from '../bolConfirmations'
import type { BolShipmentConfirmation } from '../bolConfirmations'

// Bol usually answers within a minute or two; the backend job checks every
// 5 minutes, so re-reading every 30s while pending shows the result soon
// after it lands without hammering the API.
export const BOL_CONFIRMATION_POLL_MS = 30_000

interface UseBolShipmentConfirmationsState {
  confirmations: BolShipmentConfirmation[]
  loading: boolean
  error: string | null
  refresh: () => void
}

// useBolShipmentConfirmations loads whether this order's shipment was
// confirmed on Bol. enabled=false (not a Bol order) skips the request.
export function useBolShipmentConfirmations(token: string, salesOrderId: string | null, enabled: boolean): UseBolShipmentConfirmationsState {
  const [confirmations, setConfirmations] = useState<BolShipmentConfirmation[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!enabled || !token || !salesOrderId) {
      setConfirmations([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      setConfirmations(await listBolShipmentConfirmations(token, salesOrderId))
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('orders:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [enabled, token, salesOrderId])

  useEffect(() => { load() }, [load])

  const pending = isBolConfirmationPending(confirmations)
  useEffect(() => {
    if (!pending) return
    const timer = setTimeout(() => { load() }, BOL_CONFIRMATION_POLL_MS)
    return () => clearTimeout(timer)
  }, [pending, confirmations, load])

  return { confirmations, loading, error, refresh: load }
}
