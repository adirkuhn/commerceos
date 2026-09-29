import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { TaxClass, CreateTaxClassInput, UpdateTaxClassInput } from '../types'
import * as api from '../api'

interface UseTaxClassesState {
  classes: TaxClass[]
  loading: boolean
  error: string | null
}

interface UseTaxClassesActions {
  refresh: () => void
  create: (input: CreateTaxClassInput) => Promise<TaxClass>
  update: (id: string, input: UpdateTaxClassInput) => Promise<TaxClass>
  remove: (id: string) => Promise<void>
}

export function useTaxClasses(token: string): UseTaxClassesState & UseTaxClassesActions {
  const [classes, setClasses] = useState<TaxClass[]>([])
  // true from the first render when a load is coming, so callers never
  // mistake "not fetched yet" for "empty" (e.g. a URL-addressed record
  // flashing "not found" before the list arrives).
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listTaxClasses(token)
      setClasses(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('common:status.error'))
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  // Setting a class as default silently unsets whichever class held it
  // before (see backend's UpdateTaxClassUseCase) -- mirror that locally so
  // the list doesn't show two "default" classes until the next refresh().
  const applyDefault = (list: TaxClass[], updated: TaxClass): TaxClass[] =>
    list.map(c => (c.id === updated.id ? updated : updated.is_default ? { ...c, is_default: false } : c))

  const create = useCallback(async (input: CreateTaxClassInput): Promise<TaxClass> => {
    const result = await api.createTaxClass(token, input)
    setClasses(prev => applyDefault([...prev, result], result))
    return result
  }, [token])

  const update = useCallback(async (id: string, input: UpdateTaxClassInput): Promise<TaxClass> => {
    const result = await api.updateTaxClass(token, id, input)
    setClasses(prev => applyDefault(prev, result))
    return result
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteTaxClass(token, id)
    setClasses(prev => prev.filter(c => c.id !== id))
  }, [token])

  return { classes, loading, error, refresh: load, create, update, remove }
}
