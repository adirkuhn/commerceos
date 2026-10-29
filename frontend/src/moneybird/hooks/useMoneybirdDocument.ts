import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import * as api from '../api'
import type { MoneybirdDocumentState, MoneybirdSourceType } from '../types'
import { apiErrorText } from '../../api/apiErrorText'
import { CodedApiError } from '../../api/codedApiError'

const NOT_CONNECTED = 'MONEYBIRD_NOT_CONNECTED'

export interface UseMoneybirdDocument {
  // false when the shop has no Moneybird connection -- nothing to show.
  connected: boolean
  state: MoneybirdDocumentState | null
  loading: boolean
  error: string | null
  sending: boolean
  sendError: string | null
  send: () => Promise<void>
}

// Where a quote or order stands in Moneybird, and sending it there.
export function useMoneybirdDocument(token: string, sourceType: MoneybirdSourceType, id: string): UseMoneybirdDocument {
  const [state, setState] = useState<MoneybirdDocumentState | null>(null)
  const [connected, setConnected] = useState(false)
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      setState(await api.getMoneybirdDocument(token, sourceType, id))
      setConnected(true)
    } catch (err) {
      if (err instanceof CodedApiError && err.code === NOT_CONNECTED) {
        setConnected(false)
      } else {
        setError(apiErrorText(err) ?? i18next.t('integrations:moneybird.loadFailed'))
      }
    } finally {
      setLoading(false)
    }
  }, [token, sourceType, id])

  useEffect(() => { void load() }, [load])

  const send = useCallback(async () => {
    setSending(true)
    setSendError(null)
    try {
      const document = await api.sendToMoneybird(token, sourceType, id)
      setState(prev => prev ? { ...prev, document } : prev)
    } catch (err) {
      setSendError(apiErrorText(err) ?? i18next.t('integrations:moneybird.sendFailed'))
      await load()
    } finally {
      setSending(false)
    }
  }, [token, sourceType, id, load])

  return { connected, state, loading, error, sending, sendError, send }
}
