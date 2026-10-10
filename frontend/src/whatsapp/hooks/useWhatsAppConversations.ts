import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { WhatsAppConversation } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseWhatsAppConversationsState {
  conversations: WhatsAppConversation[]
  loading: boolean
  error: string | null
}

interface UseWhatsAppConversationsActions {
  refresh: () => void
  deleteConversation: (phoneE164: string) => Promise<void>
}

export function useWhatsAppConversations(token: string): UseWhatsAppConversationsState & UseWhatsAppConversationsActions {
  const [conversations, setConversations] = useState<WhatsAppConversation[]>([])
  // true from the first render when a load is coming -- see useConnections.
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listWhatsAppConversations(token)
      setConversations(data)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('common:status.error'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const deleteConversation = useCallback(async (phoneE164: string): Promise<void> => {
    await api.deleteWhatsAppConversation(token, phoneE164)
    setConversations(prev => prev.filter(c => c.phone_e164 !== phoneE164))
  }, [token])

  return { conversations, loading, error, refresh: load, deleteConversation }
}
