import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { ExternalCategoryLink } from '../types'
import * as api from '../api'

interface UseExternalCategoryLinksState {
  links: ExternalCategoryLink[]
  loading: boolean
  error: string | null
}

interface UseExternalCategoryLinksActions {
  refresh: () => void
  upsert: (externalId: string, categoryId: string) => Promise<ExternalCategoryLink>
  remove: (id: string) => Promise<void>
}

export function useExternalCategoryLinks(token: string, connectionId: string): UseExternalCategoryLinksState & UseExternalCategoryLinksActions {
  const [links, setLinks] = useState<ExternalCategoryLink[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !connectionId) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listExternalCategoryLinks(token, connectionId)
      setLinks(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('integrations:unknownError'))
    } finally {
      setLoading(false)
    }
  }, [token, connectionId])

  useEffect(() => { load() }, [load])

  const upsert = useCallback(async (externalId: string, categoryId: string): Promise<ExternalCategoryLink> => {
    const result = await api.upsertExternalCategoryLink(token, connectionId, { external_id: externalId, category_id: categoryId })
    setLinks(prev => (prev.some(l => l.id === result.id) ? prev.map(l => (l.id === result.id ? result : l)) : [...prev, result]))
    return result
  }, [token, connectionId])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteExternalCategoryLink(token, connectionId, id)
    setLinks(prev => prev.filter(l => l.id !== id))
  }, [token, connectionId])

  return { links, loading, error, refresh: load, upsert, remove }
}
