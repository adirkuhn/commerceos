import { useState, useEffect, useCallback } from 'react'
import type { Warehouse, WarehouseInput } from '../types'
import * as api from '../api'

interface UseWarehousesState {
  warehouses: Warehouse[]
  loading: boolean
  error: string | null
}

interface UseWarehousesActions {
  refresh: () => void
  create: (input: WarehouseInput) => Promise<Warehouse>
  update: (id: string, input: WarehouseInput) => Promise<Warehouse>
  remove: (id: string) => Promise<void>
}

export function useWarehouses(token: string): UseWarehousesState & UseWarehousesActions {
  const [warehouses, setWarehouses] = useState<Warehouse[]>([])
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
      const data = await api.listWarehouses(token)
      setWarehouses(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: WarehouseInput): Promise<Warehouse> => {
    const warehouse = await api.createWarehouse(token, input)
    setWarehouses(prev => [...prev, warehouse])
    return warehouse
  }, [token])

  const update = useCallback(async (id: string, input: WarehouseInput): Promise<Warehouse> => {
    const warehouse = await api.updateWarehouse(token, id, input)
    setWarehouses(prev => prev.map(w => (w.id === id ? warehouse : w)))
    return warehouse
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deleteWarehouse(token, id)
    setWarehouses(prev => prev.filter(w => w.id !== id))
  }, [token])

  return { warehouses, loading, error, refresh: load, create, update, remove }
}
