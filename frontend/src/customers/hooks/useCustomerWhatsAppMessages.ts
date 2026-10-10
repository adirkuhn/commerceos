import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { WhatsAppMessage } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseCustomerWhatsAppMessagesState {
  messages: WhatsAppMessage[]
  loading: boolean
  error: string | null
}

// customerId null (a brand-new, not-yet-saved customer) means "nothing to
// fetch yet" -- same convention as useProductDetail's productId.
export function useCustomerWhatsAppMessages(token: string, customerId: string | null): UseCustomerWhatsAppMessagesState & { refresh: () => void } {
  const [messages, setMessages] = useState<WhatsAppMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !customerId) {
      setMessages([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      const data = await api.listCustomerWhatsAppMessages(token, customerId)
      setMessages(data)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('customers:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token, customerId])

  useEffect(() => { load() }, [load])

  return { messages, loading, error, refresh: load }
}
