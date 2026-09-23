import { useState, useEffect, useCallback } from 'react'
import type { WhatsAppConversation } from '../types'
import * as api from '../api'

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
      setError(e instanceof Error ? e.message : 'Unknown error')
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
