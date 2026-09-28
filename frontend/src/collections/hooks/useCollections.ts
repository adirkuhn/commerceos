import { useCallback, useEffect, useState } from 'react'
import * as api from '../api'
import type { Collection, CollectionInput } from '../types'

export function useCollections(token: string) {
  const [collections, setCollections] = useState<Collection[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      setCollections((await api.listCollections(token)).items)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load collections')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const save = async (id: string | null, input: CollectionInput, productIds?: string[]): Promise<Collection> => {
    let saved = id ? await api.updateCollection(token, id, input) : await api.createCollection(token, input)
    if (input.type === 'manual' && productIds) saved = await api.setCollectionProducts(token, saved.id, productIds)
    await refresh()
    return saved
  }

  const remove = async (id: string) => {
    await api.deleteCollection(token, id)
    setCollections(cs => cs.filter(c => c.id !== id))
  }

  return { collections, loading, error, save, remove }
}
