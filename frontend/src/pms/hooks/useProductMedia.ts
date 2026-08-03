import { useState, useEffect, useCallback } from 'react'
import type { Media } from '../types'
import * as api from '../api'

interface UseProductMediaState {
  media: Media[]
  loading: boolean
  error: string | null
}

interface UseProductMediaActions {
  refresh: () => void
  upload: (file: File, altText?: string) => Promise<Media>
  update: (id: string, altText: string) => Promise<Media>
  remove: (id: string) => Promise<void>
}

// productId is null while creating a new product (no product to attach
// images to until it's been saved and has a real id).
export function useProductMedia(token: string, productId: string | null): UseProductMediaState & UseProductMediaActions {
  const [media, setMedia] = useState<Media[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !productId) {
      setMedia([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      const data = await api.listProductMedia(token, productId)
      setMedia(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, productId])

  useEffect(() => { load() }, [load])

  const upload = useCallback(async (file: File, altText?: string): Promise<Media> => {
    if (!productId) throw new Error('Save the product before uploading images')
    const uploaded = await api.uploadProductMedia(token, productId, file, altText)
    setMedia(prev => [...prev, uploaded])
    return uploaded
  }, [token, productId])

  const update = useCallback(async (id: string, altText: string): Promise<Media> => {
    const updated = await api.updateMedia(token, id, altText)
    setMedia(prev => prev.map(m => m.id === id ? updated : m))
    return updated
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteMedia(token, id)
    setMedia(prev => prev.filter(m => m.id !== id))
  }, [token])

  return { media, loading, error, refresh: load, upload, update, remove }
}
