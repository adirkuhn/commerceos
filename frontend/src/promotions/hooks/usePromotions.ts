import { useState, useEffect, useCallback } from 'react'
import type { Promotion, PromotionDetail, PromotionInput } from '../types'
import * as api from '../api'

interface UsePromotionsState {
  promotions: Promotion[]
  loading: boolean
  error: string | null
}

interface UsePromotionsActions {
  refresh: () => void
  create: (input: PromotionInput) => Promise<PromotionDetail>
  update: (id: string, input: PromotionInput) => Promise<PromotionDetail>
  remove: (id: string) => Promise<void>
}

export function usePromotions(token: string): UsePromotionsState & UsePromotionsActions {
  const [promotions, setPromotions] = useState<Promotion[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listPromotions(token)
      setPromotions(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: PromotionInput): Promise<PromotionDetail> => {
    const promotion = await api.createPromotion(token, input)
    setPromotions(prev => [...prev, promotion])
    return promotion
  }, [token])

  const update = useCallback(async (id: string, input: PromotionInput): Promise<PromotionDetail> => {
    const promotion = await api.updatePromotion(token, id, input)
    setPromotions(prev => prev.map(p => (p.id === id ? promotion : p)))
    return promotion
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deletePromotion(token, id)
    setPromotions(prev => prev.filter(p => p.id !== id))
  }, [token])

  return { promotions, loading, error, refresh: load, create, update, remove }
}
