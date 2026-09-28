import { useCallback, useEffect, useState } from 'react'
import * as api from './api'
import type { Theme } from './types'

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
      setError(err instanceof Error ? err.message : 'Failed to load themes')
    } finally {
      setLoading(false)
    }
  }, [token, salesChannelId])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const create = async (name: string) => {
    const t = await api.createTheme(token, salesChannelId, name)
    await refresh()
    return t
  }

  const remove = async (id: string) => {
    await api.deleteTheme(token, id)
    setThemes(ts => ts.filter(t => t.id !== id))
  }

  return { themes, loading, error, refresh, create, remove }
}
