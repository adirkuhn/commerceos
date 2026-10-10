import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { WebhookForwardRule, WebhookForwardDelivery } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseWebhookForwardRulesState {
  rules: WebhookForwardRule[]
  deliveries: WebhookForwardDelivery[]
  loading: boolean
  error: string | null
}

interface UseWebhookForwardRulesActions {
  refresh: () => void
  create: (input: { provider: string; topic?: string; target_url: string }) => Promise<void>
  remove: (id: string) => Promise<void>
}

// Backs the Integrations page's "Webhook forwarding" panel -- the tenant's
// own fan-out rules (provider/topic -> target URL) plus a recent-delivery
// log so an admin can see whether a rule actually fired. See
// backend/domain/webhooks.ForwardRule/ForwardDelivery.
export function useWebhookForwardRules(token: string): UseWebhookForwardRulesState & UseWebhookForwardRulesActions {
  const [rules, setRules] = useState<WebhookForwardRule[]>([])
  const [deliveries, setDeliveries] = useState<WebhookForwardDelivery[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const [rulesResult, deliveriesResult] = await Promise.all([
        api.listWebhookForwardRules(token),
        api.listWebhookForwardDeliveries(token, 50),
      ])
      setRules(rulesResult)
      setDeliveries(deliveriesResult)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('integrations:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: { provider: string; topic?: string; target_url: string }): Promise<void> => {
    const rule = await api.createWebhookForwardRule(token, input)
    setRules(prev => [rule, ...prev])
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteWebhookForwardRule(token, id)
    setRules(prev => prev.filter(r => r.id !== id))
  }, [token])

  return { rules, deliveries, loading, error, refresh: load, create, remove }
}
