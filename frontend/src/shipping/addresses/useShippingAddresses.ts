import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import * as shippingApi from '../api'
import type { ShippingAddresses, UpdateShippingAddressesInput } from '../types'
import { apiErrorText } from '../../api/apiErrorText'

export interface UseShippingAddresses {
  addresses: ShippingAddresses | null
  loading: boolean
  error: string | null
  // Replaces both addresses; rejects on failure so the caller can show why.
  save: (input: UpdateShippingAddressesInput) => Promise<void>
}

// The tenant's label sender and return address (GET/PUT /shipping/addresses).
export function useShippingAddresses(token: string): UseShippingAddresses {
  const [addresses, setAddresses] = useState<ShippingAddresses | null>(null)
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError(null)
    shippingApi.getShippingAddresses(token)
      .then(result => { if (!cancelled) setAddresses(result) })
      .catch(err => { if (!cancelled) setError(apiErrorText(err) ?? i18next.t('shipping:addresses.loadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token])

  const save = useCallback(async (input: UpdateShippingAddressesInput) => {
    setAddresses(await shippingApi.updateShippingAddresses(token, input))
  }, [token])

  return { addresses, loading, error, save }
}
