import { useState, useEffect, useCallback } from 'react'
import type { PageSlug, SaveThemePagePayload, ThemePage } from '../types'
import * as api from '../api'

interface UseThemePageState {
  page: ThemePage | null
  loading: boolean
  saving: boolean
  error: string | null
}

interface UseThemePageActions {
  save: (payload: SaveThemePagePayload) => Promise<ThemePage>
}

export function useThemePage(token: string, channelId: string, slug: PageSlug): UseThemePageState & UseThemePageActions {
  const [page, setPage] = useState<ThemePage | null>(null)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !channelId) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.getThemePage(token, channelId, slug)
      setPage(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, channelId, slug])

  useEffect(() => { load() }, [load])

  const save = useCallback(async (payload: SaveThemePagePayload): Promise<ThemePage> => {
    setSaving(true)
    setError(null)
    try {
      const saved = await api.saveThemePage(token, channelId, slug, payload)
      setPage(saved)
      return saved
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
      throw e
    } finally {
      setSaving(false)
    }
  }, [token, channelId, slug])

  return { page, loading, saving, error, save }
}
