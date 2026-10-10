import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { ProviderWebhook } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseAllProviderWebhooksState {
  webhooks: ProviderWebhook[]
  loading: boolean
  error: string | null
}

interface UseAllProviderWebhooksActions {
  refresh: () => void
  remove: (connectionId: string, webhookId: string) => Promise<void>
}

// Tenant-wide "what's actually subscribed right now" view -- aggregates
// every WooCommerce/MyParcel connection's own provider-side subscriptions
// into one list, instead of opening each connection's settings page one at
// a time. See backend/application/integrations.ListAllProviderWebhooksUseCase.
export function useAllProviderWebhooks(token: string): UseAllProviderWebhooksState & UseAllProviderWebhooksActions {
  const [webhooks, setWebhooks] = useState<ProviderWebhook[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      setWebhooks(await api.listAllProviderWebhooks(token))
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('integrations:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const remove = useCallback(async (connectionId: string, webhookId: string): Promise<void> => {
    await api.deleteProviderWebhook(token, connectionId, webhookId)
    setWebhooks(prev => prev.filter(w => !(w.connection_id === connectionId && w.id === webhookId)))
  }, [token])

  return { webhooks, loading, error, refresh: load, remove }
}
