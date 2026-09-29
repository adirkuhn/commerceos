import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { Promotion, PromotionDetail, PromotionInput, CreatePromotionResult } from '../types'
import * as api from '../api'

interface UsePromotionsState {
  promotions: Promotion[]
  loading: boolean
  error: string | null
}

interface UsePromotionsActions {
  refresh: () => void
  create: (input: PromotionInput) => Promise<CreatePromotionResult>
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
      setError(e instanceof Error ? e.message : i18next.t('common:status.error'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: PromotionInput): Promise<CreatePromotionResult> => {
    const result = await api.createPromotion(token, input)
    setPromotions(prev => [...prev, result.promotion])
    return result
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
