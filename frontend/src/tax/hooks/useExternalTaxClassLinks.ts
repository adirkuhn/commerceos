import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { ExternalTaxClassLink, UpsertExternalTaxClassLinkInput } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseExternalTaxClassLinksState {
  links: ExternalTaxClassLink[]
  loading: boolean
  error: string | null
}

interface UseExternalTaxClassLinksActions {
  refresh: () => void
  upsert: (input: UpsertExternalTaxClassLinkInput) => Promise<ExternalTaxClassLink>
  remove: (id: string) => Promise<void>
}

export function useExternalTaxClassLinks(token: string): UseExternalTaxClassLinksState & UseExternalTaxClassLinksActions {
  const [links, setLinks] = useState<ExternalTaxClassLink[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listExternalTaxClassLinks(token)
      setLinks(data)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('common:status.error'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const upsert = useCallback(async (input: UpsertExternalTaxClassLinkInput): Promise<ExternalTaxClassLink> => {
    const result = await api.upsertExternalTaxClassLink(token, input)
    setLinks(prev => (prev.some(l => l.id === result.id) ? prev.map(l => (l.id === result.id ? result : l)) : [...prev, result]))
    return result
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteExternalTaxClassLink(token, id)
    setLinks(prev => prev.filter(l => l.id !== id))
  }, [token])

  return { links, loading, error, refresh: load, upsert, remove }
}
