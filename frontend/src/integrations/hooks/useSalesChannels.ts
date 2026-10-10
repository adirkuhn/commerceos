import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { SalesChannel } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

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
  // true from the first render when a load is coming, so callers never
  // mistake "not fetched yet" for "empty" (e.g. a URL-addressed record
  // flashing "not found" before the list arrives).
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listSalesChannels(token)
      setChannels(data)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('integrations:unknownError'))
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
