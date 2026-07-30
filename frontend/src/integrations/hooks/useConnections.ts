import { useState, useEffect, useCallback } from 'react'
import type { Connection } from '../types'
import * as api from '../api'

interface UseConnectionsState {
  connections: Connection[]
  loading: boolean
  error: string | null
}

interface UseConnectionsActions {
  refresh: () => void
  disconnect: (id: string) => Promise<void>
}

export function useConnections(token: string): UseConnectionsState & UseConnectionsActions {
  const [connections, setConnections] = useState<Connection[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listConnections(token)
      setConnections(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const disconnect = useCallback(async (id: string): Promise<void> => {
    await api.disconnectConnection(token, id)
    setConnections(prev => prev.filter(c => c.id !== id))
  }, [token])

  return { connections, loading, error, refresh: load, disconnect }
}
