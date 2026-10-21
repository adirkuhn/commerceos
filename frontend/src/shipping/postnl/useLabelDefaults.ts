import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import * as shippingApi from '../api'
import type { ShippingConnectionSettings } from '../types'
import { settingsWithLabelDefaults, type LabelDefaults } from './labelDefaults'
import { apiErrorText } from '../../api/apiErrorText'

export interface UseLabelDefaults {
  // null until loaded, and when the connection has no saved settings yet.
  settings: ShippingConnectionSettings | null
  loading: boolean
  error: string | null
  // Saves the full settings object with these fields replaced; rejects on
  // failure so the caller can show why.
  save: (defaults: LabelDefaults) => Promise<void>
}

// A connection's label defaults (GET/PUT /shipping/connections/{id}/settings).
export function useLabelDefaults(token: string, connectionId: string): UseLabelDefaults {
  const [settings, setSettings] = useState<ShippingConnectionSettings | null>(null)
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError(null)
    shippingApi.getShippingSettings(token, connectionId)
      .then(result => { if (!cancelled) setSettings(result.settings) })
      .catch(err => { if (!cancelled) setError(apiErrorText(err) ?? i18next.t('shipping:myparcel.loadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, connectionId])

  const save = useCallback(async (defaults: LabelDefaults) => {
    setSettings(await shippingApi.updateShippingSettings(token, connectionId, settingsWithLabelDefaults(settings, defaults)))
  }, [token, connectionId, settings])

  return { settings, loading, error, save }
}
