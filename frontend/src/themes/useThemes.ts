import { useCallback, useEffect, useState } from 'react'
import i18next from 'i18next'
import * as api from './api'
import type { Theme, ThemePreset } from './types'

export function useThemes(token: string, salesChannelId: string) {
  const [themes, setThemes] = useState<Theme[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      setThemes((await api.listThemes(token, salesChannelId)).items)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : i18next.t('themes:errors.load'))
    } finally {
      setLoading(false)
    }
  }, [token, salesChannelId])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const create = async (name: string, preset: ThemePreset = 'default') => {
    const t = await api.createTheme(token, salesChannelId, name, preset)
    await refresh()
    return t
  }

  const remove = async (id: string) => {
    await api.deleteTheme(token, id)
    setThemes(ts => ts.filter(t => t.id !== id))
  }

  /** Rollback: makes this published version (and its theme) live. */
  const makeLive = async (themeId: string, versionId: string) => {
    await api.rollbackTheme(token, themeId, versionId)
    await refresh()
  }

  return { themes, loading, error, refresh, create, remove, makeLive }
}
