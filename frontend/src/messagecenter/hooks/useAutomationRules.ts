import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { AutomationRule, CreateAutomationRuleInput, UpdateAutomationRuleInput } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseAutomationRulesState {
  rules: AutomationRule[]
  loading: boolean
  error: string | null
}

interface UseAutomationRulesActions {
  refresh: () => void
  create: (input: CreateAutomationRuleInput) => Promise<void>
  update: (id: string, input: UpdateAutomationRuleInput) => Promise<void>
  remove: (id: string) => Promise<void>
}

export function useAutomationRules(token: string): UseAutomationRulesState & UseAutomationRulesActions {
  const [rules, setRules] = useState<AutomationRule[]>([])
  // true from the first render when a load is coming -- see useConnections.
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listAutomationRules(token)
      setRules(data)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('common:status.error'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: CreateAutomationRuleInput): Promise<void> => {
    const created = await api.createAutomationRule(token, input)
    setRules(prev => [created, ...prev])
  }, [token])

  const update = useCallback(async (id: string, input: UpdateAutomationRuleInput): Promise<void> => {
    const updated = await api.updateAutomationRule(token, id, input)
    setRules(prev => prev.map(r => (r.id === id ? updated : r)))
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteAutomationRule(token, id)
    setRules(prev => prev.filter(r => r.id !== id))
  }, [token])

  return { rules, loading, error, refresh: load, create, update, remove }
}
