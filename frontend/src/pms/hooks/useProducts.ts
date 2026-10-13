import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { Product, CreateProductPayload, UpdateProductPayload } from '../types'
import * as api from '../api'
import type { ProductStockFilter } from '../api'
import { notifyProductsChanged, subscribeProductsChanged } from '../productsBus'
import { apiErrorText } from '../../api/apiErrorText'

interface UseProductsState {
  products: Product[]
  total: number
  loading: boolean
  error: string | null
}

interface UseProductsActions {
  refresh: () => void
  create: (payload: CreateProductPayload) => Promise<Product>
  update: (id: string, payload: UpdateProductPayload) => Promise<Product>
  remove: (id: string) => Promise<void>
}

export function useProducts(token: string, params: { page?: number; perPage?: number; search?: string; stock?: ProductStockFilter } = {}): UseProductsState & UseProductsActions {
  const { page, perPage, search, stock } = params
  const [products, setProducts] = useState<Product[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const result = await api.listProducts(token, { page, perPage, search, stock })
      setProducts(result.products)
      setTotal(result.total)
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('common:status.error'))
    } finally {
      setLoading(false)
    }
  }, [token, page, perPage, search, stock])

  useEffect(() => { load() }, [load])

  // Other instances of this hook (e.g. the Sidebar badge) — or code that
  // mutates products without going through this hook at all (bulk
  // WooCommerce import/delete) — announce changes here so this instance
  // stays in sync instead of showing a stale count.
  useEffect(() => subscribeProductsChanged(load), [load])

  // Mutations refetch the current page rather than patching the array
  // locally — with server-side pagination/totals, splicing in place would
  // drift from what the server actually holds (wrong page, stale total).
  const create = useCallback(async (payload: CreateProductPayload): Promise<Product> => {
    const product = await api.createProduct(token, payload)
    await load()
    notifyProductsChanged()
    return product
  }, [token, load])

  const update = useCallback(async (id: string, payload: UpdateProductPayload): Promise<Product> => {
    const updated = await api.updateProduct(token, id, payload)
    await load()
    return updated
  }, [token, load])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteProduct(token, id)
    await load()
    notifyProductsChanged()
  }, [token, load])

  return { products, total, loading, error, refresh: load, create, update, remove }
}
