import { useState, useEffect, useCallback } from 'react'
import type { SalesChannel } from '../types'
import * as api from '../api'

interface UseSalesChannelsState {
  channels: SalesChannel[]
  loading: boolean
  error: string | null
}

interface UseSalesChannelsActions {
  refresh: () => void
  create: (payload: api.SalesChannelPayload) => Promise<SalesChannel>
  update: (id: string, payload: api.UpdateSalesChannelPayload) => Promise<SalesChannel>
  remove: (id: string) => Promise<void>
}

export function useSalesChannels(token: string): UseSalesChannelsState & UseSalesChannelsActions {
  const [channels, setChannels] = useState<SalesChannel[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listSalesChannels(token)
      setChannels(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (payload: api.SalesChannelPayload): Promise<SalesChannel> => {
    const channel = await api.createSalesChannel(token, payload)
    setChannels(prev => [...prev, channel])
    return channel
  }, [token])

  const update = useCallback(async (id: string, payload: api.UpdateSalesChannelPayload): Promise<SalesChannel> => {
    const channel = await api.updateSalesChannel(token, id, payload)
    setChannels(prev => prev.map(c => (c.id === id ? channel : c)))
    return channel
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteSalesChannel(token, id)
    setChannels(prev => prev.filter(c => c.id !== id))
  }, [token])

  return { channels, loading, error, refresh: load, create, update, remove }
}
