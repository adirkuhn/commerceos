import { useState, useEffect, useCallback } from 'react'
import type { Category, CreateCategoryPayload } from '../types'
import * as api from '../api'

interface UseCategoriesState {
  categories: Category[]
  loading: boolean
  error: string | null
}

interface UseCategoriesActions {
  refresh: () => void
  create: (payload: CreateCategoryPayload) => Promise<Category>
}

export function useCategories(token: string): UseCategoriesState & UseCategoriesActions {
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listCategories(token)
      setCategories(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (payload: CreateCategoryPayload): Promise<Category> => {
    const category = await api.createCategory(token, payload)
    setCategories(prev => [...prev, category])
    return category
  }, [token])

  return { categories, loading, error, refresh: load, create }
}
