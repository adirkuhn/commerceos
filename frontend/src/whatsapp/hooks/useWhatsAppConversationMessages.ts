import { useState, useEffect, useCallback } from 'react'
import type { WhatsAppConversationMessage } from '../types'
import * as api from '../api'

interface UseWhatsAppConversationMessagesState {
  messages: WhatsAppConversationMessage[]
  loading: boolean
  error: string | null
}

// phoneE164 null (nothing selected yet) means "nothing to fetch" -- same
// convention as useProductDetail's productId.
export function useWhatsAppConversationMessages(token: string, phoneE164: string | null): UseWhatsAppConversationMessagesState & { refresh: () => void } {
  const [messages, setMessages] = useState<WhatsAppConversationMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !phoneE164) {
      setMessages([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      const data = await api.listWhatsAppConversationMessages(token, phoneE164)
      setMessages(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, phoneE164])

  useEffect(() => { load() }, [load])

  return { messages, loading, error, refresh: load }
}
