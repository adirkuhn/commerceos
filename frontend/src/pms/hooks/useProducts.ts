import { useState, useEffect, useCallback } from 'react'
import type { Product, CreateProductPayload, UpdateProductPayload } from '../types'
import * as api from '../api'

interface UseProductsState {
  products: Product[]
  loading: boolean
  error: string | null
}

interface UseProductsActions {
  refresh: () => void
  create: (payload: CreateProductPayload) => Promise<Product>
  update: (id: string, payload: UpdateProductPayload) => Promise<Product>
  remove: (id: string) => Promise<void>
}

export function useProducts(token: string): UseProductsState & UseProductsActions {
  const [products, setProducts] = useState<Product[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listProducts(token)
      setProducts(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (payload: CreateProductPayload): Promise<Product> => {
    const product = await api.createProduct(token, payload)
    setProducts(prev => [...prev, product])
    return product
  }, [token])

  const update = useCallback(async (id: string, payload: UpdateProductPayload): Promise<Product> => {
    const updated = await api.updateProduct(token, id, payload)
    setProducts(prev => prev.map(p => p.id === id ? updated : p))
    return updated
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteProduct(token, id)
    setProducts(prev => prev.filter(p => p.id !== id))
  }, [token])

  return { products, loading, error, refresh: load, create, update, remove }
}
