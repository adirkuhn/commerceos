import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import * as shippingApi from './api'
import type { ShippingConnection, ShippingConnections } from './types'
import { apiErrorText } from '../api/apiErrorText'

const EMPTY: ShippingConnections = { connections: [], default_connection_id: null, configured_connection_id: null }

export interface UseShippingConnections {
  // Oldest first; only connections that can create a label.
  connections: ShippingConnection[]
  // What packing preselects (the merchant's pick, else the oldest).
  defaultConnection: ShippingConnection | null
  // The merchant's explicit pick; null = automatic (oldest).
  configuredConnectionId: string | null
  loading: boolean
  error: string | null
  // Saves the merchant's pick (null = automatic); rejects on failure so
  // the caller can show why.
  saveDefault: (connectionId: string | null) => Promise<void>
}

// The tenant's shipping connections and packing's default one -- shared by
// every label flow and the default-provider setting.
export function useShippingConnections(token: string): UseShippingConnections {
  const [data, setData] = useState<ShippingConnections>(EMPTY)
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError(null)
    shippingApi.listShippingConnections(token)
      .then(result => { if (!cancelled) setData(result) })
      .catch(err => { if (!cancelled) setError(apiErrorText(err) ?? i18next.t('shipping:provider.loadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token])

  const saveDefault = useCallback(async (connectionId: string | null) => {
    setData(await shippingApi.updateShippingPreferences(token, connectionId))
  }, [token])

  const defaultConnection = data.connections.find(c => c.id === data.default_connection_id) ?? null
  return {
    connections: data.connections, defaultConnection, configuredConnectionId: data.configured_connection_id,
    loading, error, saveDefault,
  }
}
