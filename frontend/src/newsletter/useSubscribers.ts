import i18next from 'i18next'
import { useCallback, useEffect, useState } from 'react'
import * as api from './api'
import type { Subscriber, SubscriberQuery } from './types'

export function useSubscribers(token: string, query: SubscriberQuery) {
  const [subscribers, setSubscribers] = useState<Subscriber[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const { status, search, page, perPage } = query

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.listSubscribers(token, { status, search, page, perPage })
      setSubscribers(res.items)
      setTotal(res.total)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : i18next.t('newsletter:list.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [token, status, search, page, perPage])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const erase = async (id: string) => {
    await api.eraseSubscriber(token, id)
    await refresh()
  }

  return { subscribers, total, loading, error, erase }
}
