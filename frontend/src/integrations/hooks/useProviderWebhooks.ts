import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { ProviderWebhook } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseProviderWebhooksState {
  webhooks: ProviderWebhook[]
  loading: boolean
  error: string | null
}

interface UseProviderWebhooksActions {
  refresh: () => void
  create: (topic: string) => Promise<void>
  remove: (webhookId: string) => Promise<void>
}

// Backs the per-connection "Provider webhook subscriptions" panel --
// what's actually registered at WooCommerce/MyParcel, as opposed to
// useWebhookForwardRules' CommerceOS-side fan-out configuration. See
// backend/domain/integrations.ProviderWebhook.
export function useProviderWebhooks(token: string, connectionId: string): UseProviderWebhooksState & UseProviderWebhooksActions {
  const [webhooks, setWebhooks] = useState<ProviderWebhook[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !connectionId) return
    setLoading(true)
    setError(null)
    try {
      setWebhooks(await api.listProviderWebhooks(token, connectionId))
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('integrations:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token, connectionId])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (topic: string): Promise<void> => {
    const webhook = await api.createProviderWebhook(token, connectionId, topic)
    setWebhooks(prev => [webhook, ...prev])
  }, [token, connectionId])

  const remove = useCallback(async (webhookId: string): Promise<void> => {
    await api.deleteProviderWebhook(token, connectionId, webhookId)
    setWebhooks(prev => prev.filter(w => w.id !== webhookId))
  }, [token, connectionId])

  return { webhooks, loading, error, refresh: load, create, remove }
}
