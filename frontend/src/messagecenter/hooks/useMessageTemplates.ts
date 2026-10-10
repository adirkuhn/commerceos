import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { MessageTemplate, CreateMessageTemplateInput, UpdateMessageTemplateInput } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseMessageTemplatesState {
  templates: MessageTemplate[]
  loading: boolean
  error: string | null
}

interface UseMessageTemplatesActions {
  refresh: () => void
  create: (input: CreateMessageTemplateInput) => Promise<void>
  update: (id: string, input: UpdateMessageTemplateInput) => Promise<void>
  remove: (id: string) => Promise<void>
}

export function useMessageTemplates(token: string): UseMessageTemplatesState & UseMessageTemplatesActions {
  const [templates, setTemplates] = useState<MessageTemplate[]>([])
  // true from the first render when a load is coming -- see useConnections.
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listMessageTemplates(token)
      setTemplates(data)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('common:status.error'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: CreateMessageTemplateInput): Promise<void> => {
    const created = await api.createMessageTemplate(token, input)
    setTemplates(prev => [created, ...prev])
  }, [token])

  const update = useCallback(async (id: string, input: UpdateMessageTemplateInput): Promise<void> => {
    const updated = await api.updateMessageTemplate(token, id, input)
    setTemplates(prev => prev.map(t => (t.id === id ? updated : t)))
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteMessageTemplate(token, id)
    setTemplates(prev => prev.filter(t => t.id !== id))
  }, [token])

  return { templates, loading, error, refresh: load, create, update, remove }
}
