import { useState, useEffect, useCallback } from 'react'
import type { Review } from '../types'
import * as api from '../api'

interface UseReviewsState {
  reviews: Review[]
  loading: boolean
  error: string | null
}

interface UseReviewsActions {
  refresh: () => void
  remove: (id: string) => Promise<void>
}

export function useReviews(token: string): UseReviewsState & UseReviewsActions {
  const [reviews, setReviews] = useState<Review[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listReviews(token)
      setReviews(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteReview(token, id)
    setReviews(prev => prev.filter(r => r.id !== id))
  }, [token])

  return { reviews, loading, error, refresh: load, remove }
}
