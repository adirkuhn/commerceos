import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { AIConnection } from '../types'
import * as api from '../api'
import type { ConnectAIProviderPayload } from '../api'

interface UseAIConnectionState {
  connection: AIConnection | null
  loading: boolean
  error: string | null
}

interface UseAIConnectionActions {
  refresh: () => void
  connect: (payload: ConnectAIProviderPayload) => Promise<AIConnection>
  disconnect: () => Promise<void>
}

export function useAIConnection(token: string): UseAIConnectionState & UseAIConnectionActions {
  const [connection, setConnection] = useState<AIConnection | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.getAIConnection(token)
      setConnection(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('integrations:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const connect = useCallback(async (payload: ConnectAIProviderPayload): Promise<AIConnection> => {
    const conn = await api.connectAIProvider(token, payload)
    setConnection(conn)
    return conn
  }, [token])

  const disconnect = useCallback(async (): Promise<void> => {
    await api.disconnectAIProvider(token)
    setConnection(null)
  }, [token])

  return { connection, loading, error, refresh: load, connect, disconnect }
}
