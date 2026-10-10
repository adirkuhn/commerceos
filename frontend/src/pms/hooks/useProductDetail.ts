import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { ProductDetail, CreateVariantPayload, UpdateVariantPayload, CreateSKUPayload, UpdateSKUPayload } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

interface UseProductDetailState {
  detail: ProductDetail | null
  loading: boolean
  error: string | null
}

interface UseProductDetailActions {
  refresh: () => void
  addVariant: (payload: CreateVariantPayload) => Promise<void>
  updateVariant: (id: string, payload: UpdateVariantPayload) => Promise<void>
  removeVariant: (id: string) => Promise<void>
  addSKU: (variantId: string, payload: CreateSKUPayload) => Promise<void>
  updateSKU: (id: string, payload: UpdateSKUPayload) => Promise<void>
  removeSKU: (id: string) => Promise<void>
}

// productId is null while creating a new product (no variants/skus to manage
// until the product itself has been saved and has a real id).
export function useProductDetail(token: string, productId: string | null): UseProductDetailState & UseProductDetailActions {
  const [detail, setDetail] = useState<ProductDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !productId) {
      setDetail(null)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const data = await api.getProduct(token, productId)
      setDetail(data)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('common:status.error'))
    } finally {
      setLoading(false)
    }
  }, [token, productId])

  useEffect(() => { load() }, [load])

  const addVariant = useCallback(async (payload: CreateVariantPayload) => {
    if (!productId) return
    await api.createVariant(token, productId, payload)
    await load()
  }, [token, productId, load])

  const updateVariant = useCallback(async (id: string, payload: UpdateVariantPayload) => {
    await api.updateVariant(token, id, payload)
    await load()
  }, [token, load])

  const removeVariant = useCallback(async (id: string) => {
    await api.deleteVariant(token, id)
    await load()
  }, [token, load])

  const addSKU = useCallback(async (variantId: string, payload: CreateSKUPayload) => {
    await api.createSKU(token, variantId, payload)
    await load()
  }, [token, load])

  const updateSKU = useCallback(async (id: string, payload: UpdateSKUPayload) => {
    await api.updateSKU(token, id, payload)
    await load()
  }, [token, load])

  const removeSKU = useCallback(async (id: string) => {
    await api.deleteSKU(token, id)
    await load()
  }, [token, load])

  return { detail, loading, error, refresh: load, addVariant, updateVariant, removeVariant, addSKU, updateSKU, removeSKU }
}
