import { useState, useEffect, useCallback } from 'react'
import type { WhatsAppConversation } from '../types'
import * as api from '../api'

interface UseWhatsAppConversationsState {
  conversations: WhatsAppConversation[]
  loading: boolean
  error: string | null
}

export function useWhatsAppConversations(token: string): UseWhatsAppConversationsState & { refresh: () => void } {
  const [conversations, setConversations] = useState<WhatsAppConversation[]>([])
  const [loading, setLoading] = useState(false)
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

  return { conversations, loading, error, refresh: load }
}
