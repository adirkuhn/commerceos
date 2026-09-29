import i18next from 'i18next'
import { useState, useEffect, useCallback, useMemo } from 'react'
import type { Notification } from '../types'
import * as api from '../api'
import { subscribeNotificationsChanged } from '../notificationsBus'

interface UseNotificationsState {
  notifications: Notification[]
  unreadCount: number
  loading: boolean
  error: string | null
}

interface UseNotificationsActions {
  refresh: () => void
  markRead: (id: string) => Promise<void>
  markAllRead: () => Promise<void>
}

export function useNotifications(token: string): UseNotificationsState & UseNotificationsActions {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listNotifications(token)
      setNotifications(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('ui:errors.unknown'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])
  useEffect(() => subscribeNotificationsChanged(load), [load])

  const markRead = useCallback(async (id: string): Promise<void> => {
    await api.markNotificationRead(token, id)
    setNotifications(prev => prev.map(n => (n.id === id ? { ...n, read: true } : n)))
  }, [token])

  const markAllRead = useCallback(async (): Promise<void> => {
    await api.markAllNotificationsRead(token)
    setNotifications(prev => prev.map(n => ({ ...n, read: true })))
  }, [token])

  const unreadCount = useMemo(() => notifications.filter(n => !n.read).length, [notifications])

  return { notifications, unreadCount, loading, error, refresh: load, markRead, markAllRead }
}
