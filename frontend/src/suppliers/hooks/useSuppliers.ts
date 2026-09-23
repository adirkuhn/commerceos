import { useState, useEffect, useCallback } from 'react'
import type { Supplier, SupplierInput } from '../types'
import * as api from '../api'

interface UseSuppliersState {
  suppliers: Supplier[]
  loading: boolean
  error: string | null
}

interface UseSuppliersActions {
  refresh: () => void
  create: (input: SupplierInput) => Promise<Supplier>
  update: (id: string, input: SupplierInput) => Promise<Supplier>
  remove: (id: string) => Promise<void>
}

export function useSuppliers(token: string): UseSuppliersState & UseSuppliersActions {
  const [suppliers, setSuppliers] = useState<Supplier[]>([])
  // true from the first render when a load is coming -- see useConnections.
  const [loading, setLoading] = useState(Boolean(token))
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listSuppliers(token)
      setSuppliers(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: SupplierInput): Promise<Supplier> => {
    const supplier = await api.createSupplier(token, input)
    setSuppliers(prev => [...prev, supplier])
    return supplier
  }, [token])

  const update = useCallback(async (id: string, input: SupplierInput): Promise<Supplier> => {
    const supplier = await api.updateSupplier(token, id, input)
    setSuppliers(prev => prev.map(s => (s.id === id ? supplier : s)))
    return supplier
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteSupplier(token, id)
    setSuppliers(prev => prev.filter(s => s.id !== id))
  }, [token])

  return { suppliers, loading, error, refresh: load, create, update, remove }
}
